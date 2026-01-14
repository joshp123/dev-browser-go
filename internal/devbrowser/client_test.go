package devbrowser

import (
	"strings"
	"testing"
)

func TestStartDaemonRejectsDeviceAndWindow(t *testing.T) {
	profile := "test-device-conflict"
	_, _ = StopDaemon(profile)

	window := &WindowSize{Width: 100, Height: 200}
	err := StartDaemon(profile, true, window, "Pixel 5")
	if err == nil {
		t.Fatalf("expected error for device + window")
	}
	if !strings.Contains(err.Error(), "use either") {
		t.Fatalf("unexpected error: %v", err)
	}
}
