package e2etest

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"google.golang.org/api/compute/v1"
)

const workZone = "us-west1-a"

// An E2ETest represent a set of E2E tests run by daisy
type E2ETest struct {
	WorkProject string
	GcsPath     string

	ImageTests []*ImageTest
}

// An ImageTest represent a group of E2E tests that could be invoke for a certain image
type ImageTest struct {
	TestName string
	// for a new candidate image
	TarBallPath string
	// for prod image
	ImageName    string
	ImageFamily  string
	ImageProject string

	TestBinaryPath string
}

func (t *E2ETest) CreateWorkflows(ctx context.Context) ([]*daisy.Workflow, error) {
	var ws []*daisy.Workflow
	for _, it := range t.ImageTests {
		fmt.Printf("Creating E2E Testing Workflows\n")
		w, err := createWorkflow(ctx, t.WorkProject, it)
		if err != nil {
			return nil, err
		}
		if err := w.PopulateClients(ctx); err != nil {
			return nil, fmt.Errorf("PopulateClients failed: %s", err)
		}
		if len(w.Steps) == 0 {
			return nil, nil
		}
		if t.GcsPath != "" {
			w.GCSPath = t.GcsPath
		}
		if w == nil {
			continue
		}
		//fmt.Printf("workflow step info %s", w.Steps["create-instance"])
		ws = append(ws, w)
	}
	return ws, nil
}

// To test a new candidate image, using createNewImageTestWorkfow. To test prod images using createExistImageTestWorkflow
func createWorkflow(ctx context.Context, workProject string, iTest *ImageTest) (*daisy.Workflow, error) {
	w := daisy.New()
	w.Name = "e2e-test-" + randString(5)
	w.Project = workProject

	var err error
	if iTest.TarBallPath != "" {
		// new candidate image E2E test
		fmt.Printf("Create Image Test for New Image\n")
		w, err = createNewImageTestWorkfow(w, workProject, iTest)
	} else {
		// new guest package E2E test on exist image , don't need to create Image from tarball, using prod image
		fmt.Printf("Create Image Test for Exist Image\n")
		w, err = createExistImageTestWorkflow(w, workProject, iTest, nil, false)
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func createNewImageTestWorkfow(w *daisy.Workflow, workProject string, iTest *ImageTest) (*daisy.Workflow, error) {
	iTest.ImageName = "e2etest-image"
	iTest.ImageProject = workProject
	cis := createImages(iTest.ImageName, workProject, iTest.TarBallPath)
	return createExistImageTestWorkflow(w, workProject, iTest, cis, true)
}

func createExistImageTestWorkflow(w *daisy.Workflow, workProject string, iTest *ImageTest, cis *daisy.CreateImages, isNewImage bool) (*daisy.Workflow, error) {
	var diskName = "disk"
	var instanceName = "instance"

	cds := createDisks(iTest.ImageName, workProject, iTest.ImageProject, diskName, isNewImage)
	cins := createInstances(workProject, diskName, instanceName, iTest.TestBinaryPath)
	ws := waitForInstancesSignal(instanceName)
	drs := deleteResource(diskName, iTest.ImageName, instanceName, isNewImage)

	w, err := populateSteps(w, cis, cds, cins, ws, drs)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func createImages(imageName, workProject, tarBallPath string) *daisy.CreateImages {
	ci := daisy.Image{
		Image: compute.Image{
			Name:    imageName,
			RawDisk: &compute.ImageRawDisk{Source: tarBallPath},
		},
		ImageBase: daisy.ImageBase{
			Resource: daisy.Resource{
				NoCleanup: true,
				Project:   workProject,
				RealName:  imageName,
			},
			IgnoreLicenseValidationIfForbidden: true,
		},
	}
	cis := &daisy.CreateImages{Images: []*daisy.Image{&ci}}
	return cis
}

func createDisks(imageName, workProject, imageProject, diskName string, isNewImage bool) *daisy.CreateDisks {
	cd := daisy.Disk{
		Disk: compute.Disk{
			Name:        diskName,
			Zone:        workZone,
			SourceImage: getSourceImageReference(imageName, isNewImage, imageProject),
		},
		Resource: daisy.Resource{
			Project: workProject,
		},
		IsWindows:            "false",
		SizeGb:               "20",
		FallbackToPdStandard: false,
	}
	cds := &daisy.CreateDisks{&cd}
	return cds
}

func getSourceImageReference(imageName string, isNewImage bool, imageProject string) string {
	var sourceImage string
	if isNewImage {
		// reference the new image created in last step
		sourceImage = imageName
	} else {
		// reference the image provided
		sourceImage = fmt.Sprintf("projects/%s/global/images/%s", imageProject, imageName)
	}
	return sourceImage
}

func createInstances(workProject, diskName, instanceName, testBinaryPath string) *daisy.CreateInstances {
	b, err := ioutil.ReadFile("bootstrap.sh")
	if err != nil {
		return nil
	}
	startupScriptContent := string(b)
	cin := daisy.Instance{
		Instance: compute.Instance{
			Name: instanceName,
			Zone: workZone,
			Disks: []*compute.AttachedDisk{
				{Source: diskName},
			},
		},
		InstanceBase: daisy.InstanceBase{
			Resource: daisy.Resource{
				Project: workProject,
			},
		},
		Metadata: map[string]string{
			// Bootstrap script to download test wrapper
			"startup-script": startupScriptContent,
			// Test binary used in test wrapper to identify which test to run
			"test-binary-path": testBinaryPath,
		},
	}
	cins := &daisy.CreateInstances{Instances: []*daisy.Instance{&cin}}
	return cins
}

func waitForInstancesSignal(instanceName string) *daisy.WaitForInstancesSignal {
	wfis := daisy.InstanceSignal{
		Name:    instanceName,
		SerialOutput: &daisy.SerialOutput{
			Port:         1,
			SuccessMatch: "E2ESuccess",
			FailureMatch: []string{"E2EFailed"},
			StatusMatch:  "E2EStatus",
		},
	}
	wfiss := &daisy.WaitForInstancesSignal{&wfis}
	return wfiss
}

func deleteResource(diskName, imageName, instanceName string, isNewImage bool) *daisy.DeleteResources {
	var tobeDeleteImages = []string{}
	if isNewImage {
		tobeDeleteImages = append(tobeDeleteImages, imageName)
	}
	return &daisy.DeleteResources{
		Disks:     []string{diskName},
		Images:    tobeDeleteImages,
		Instances: []string{instanceName},
	}
}

func populateSteps(w *daisy.Workflow, cis *daisy.CreateImages, cds *daisy.CreateDisks, cins *daisy.CreateInstances, ws *daisy.WaitForInstancesSignal, drs *daisy.DeleteResources) (*daisy.Workflow, error) {
	var err error

	var createImageStep *daisy.Step
	var createDiskStep *daisy.Step
	var createInstanceStep *daisy.Step
	var waitStep *daisy.Step
	var deleteStep *daisy.Step

	if cis != nil {
		createImageStep, err = w.NewStep("create-image")
		if err != nil {
			return nil, err
		}
		createImageStep.CreateImages = cis
	}

	if cds != nil {
		createDiskStep, err = w.NewStep("create-disk")
		if err != nil {
			return nil, err
		}
		createDiskStep.CreateDisks = cds
	}

	if cins != nil {
		createInstanceStep, err = w.NewStep("create-instance")
		if err != nil {
			return nil, err
		}
		createInstanceStep.CreateInstances = cins
	}

	if ws != nil {
		waitStep, err = w.NewStep("wait")
		if err != nil {
			return nil, err
		}
		waitStep.WaitForInstancesSignal = ws
	}

	if drs != nil {
		deleteStep, err = w.NewStep("delete")
		if err != nil {
			return nil, err
		}
		deleteStep.DeleteResources = drs
	}

	if createDiskStep != nil && createImageStep != nil {
		w.AddDependency(createDiskStep, createImageStep)
	}
	if createInstanceStep != nil && createDiskStep != nil {
		w.AddDependency(createInstanceStep, createDiskStep)
	}
	if waitStep != nil && createInstanceStep != nil {
		w.AddDependency(waitStep, createInstanceStep)
	}
	if deleteStep != nil && waitStep != nil {
		w.AddDependency(deleteStep, waitStep)
	}
	return w, nil
}

func randString(n int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	letters := "bdghjlmnpqrstvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[gen.Int63()%int64(len(letters))]
	}
	return string(b)
}
