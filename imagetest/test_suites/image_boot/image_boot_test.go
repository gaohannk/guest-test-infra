package image_boot

import (
	"syscall"
	"testing"
	"os"
)

func TestGuestBoot(t *testing.T) {
	err := syscall.Uname(&syscall.Utsname{})

	if err != nil {
		t.Fatalf("couldn't get system information, image boot failed")
	}

}

func TestGuestReboot(t *testing.T) {
	_, err := os.Stat("boot")
	if os.IsNotExist(err) {
		// first boot
		if _, err := os.Create("boot"); err != nil {
			t.Fatal("fail to create file when first boot")
			return
		}
		return
	}
	// second boot
	err = syscall.Uname(&syscall.Utsname{})

	if err != nil {
		t.Fatalf("couldn't get system information, image reboot failed")
	}
}
