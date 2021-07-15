package imageboot

import (
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

const shutdownTime = 110 // about 2 minutes

// TestGuestShutdownScript test that shutdown scripts can run for around two minutes
func TestGuestShutdownScript(t *testing.T) {
	// second boot
	bytes, err := ioutil.ReadFile("/shutdown.txt")
	if err != nil {
		t.Fatalf("error reading file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	if len(lines) < shutdownTime {
		t.Fatalf("shut down time less than %d seconds.", shutdownTime)
	}
}

func TestStartAndImmediateShutdown(t *testing.T) {
	time.Sleep(5 * time.Second)
}
