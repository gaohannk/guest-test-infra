package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
	"guest-test-infra/test_manager/e2etest"
)

var (
	workProject    = flag.String("work_project", "", "project to perform the work in, passed to Daisy as workflow project, will override WorkProject in template")
	gcsPath        = flag.String("gcs_path", "", "GCS bucket to use, overrides what is set in workflow")

	testName       = flag.String("test_name", "", "the E2E test name used for logging and daisy")
	tarballPath    = flag.String("tarball_path", "", "a list of image tarball path that want to test, split by ',', at least one provided")
	imageName      = flag.String("image_name", "", "a list of image that target image want to test, split by ',', at least one provided")
	imageFamily    = flag.String("image_family", "", "a list of image family that target image want to test belonged to, split by ',', at least one provided")
	imageProject   = flag.String("image_project", "", "a list of project that target image want to test located in, split by ',', at least one provided")
	testBinaryPath = flag.String("test_binary_path", "", "a list of binary path that going to execute in VM, plit by ',', at least one provided")
)

func main() {
	flag.Parse()

	if *tarballPath != "" && *imageName != "" {
		fmt.Println("Cannot set both -tarball_path and -image_name")
		os.Exit(1)
	}

	imageTests := parseImageTestArg()
	e2eTest := e2etest.E2ETest{
		WorkProject: *workProject,
		GcsPath:     *gcsPath,
		ImageTests:  imageTests,
	}

	var ws []*daisy.Workflow
	var errs []error
	var err error

	ctx := context.Background()
	ws, err = e2eTest.CreateWorkflows(ctx)
	if err != nil {
		createWorkflowErr := fmt.Errorf("workflow creation error: %s", err)
		fmt.Println(createWorkflowErr)
		errs = append(errs, createWorkflowErr)
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(ws)+len(errs))
	for _, w := range ws {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func(w *daisy.Workflow) {
			select {
			case <-c:
				fmt.Printf("\nCtrl-C caught, sending cancel signal to %q...\n", w.Name)
				close(w.Cancel)
				errors <- fmt.Errorf("workflow %q was canceled", w.Name)
			case <-w.Cancel:
			}
		}(w)

		wg.Add(1)
		go func(w *daisy.Workflow) {
			defer wg.Done()
			fmt.Printf("\n[E2E Test] Running workflow %q\n", w.Name)
			if err := w.Run(ctx); err != nil {
				errors <- fmt.Errorf("%s: %v", w.Name, err)
				return
			}
			fmt.Printf("\n[E2E Test] Workflow %q finished\n", w.Name)
		}(w)
	}
	wg.Wait()

	checkError(errors)

}

func checkError(errors chan error) {
	select {
	case err := <-errors:
		fmt.Fprintln(os.Stderr, "\n[E2E Test] Errors in one or more workflows:")
		fmt.Fprintln(os.Stderr, " ", err)
		for {
			select {
			case err := <-errors:
				fmt.Fprintln(os.Stderr, " ", err)
				continue
			default:
				os.Exit(1)
			}
		}
	default:
		return
	}
}
func parseImageTestArg() []*e2etest.ImageTest {
	var imageTests []*e2etest.ImageTest
	tarballPath := strings.Split(*tarballPath, ",")
	testName := strings.Split(*testName, ",")
	imageName := strings.Split(*imageName, ",")
	imageProject := strings.Split(*imageProject, ",")
	imageFamily := strings.Split(*imageFamily, ",")
	testBinaryPath := strings.Split(*testBinaryPath, ",")
	for idx, _ := range imageName {
		var imageTest = &e2etest.ImageTest{
			testName[idx],
			tarballPath[idx],
			imageName[idx],
			imageFamily[idx],
			imageProject[idx],
			testBinaryPath[idx],
		}
		imageTests = append(imageTests, imageTest)
	}
	return imageTests
}
