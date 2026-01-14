package devbrowser

import (
	"testing"

	"github.com/playwright-community/playwright-go"
)

func TestDeviceWindowSize(t *testing.T) {
	if got := deviceWindowSize(nil); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
	if got := deviceWindowSize(&playwright.DeviceDescriptor{}); got != nil {
		t.Fatalf("expected nil for missing viewport, got %#v", got)
	}
	desc := &playwright.DeviceDescriptor{
		Viewport: &playwright.Size{Width: 320, Height: 640},
	}
	got := deviceWindowSize(desc)
	if got == nil || got.Width != 320 || got.Height != 640 {
		t.Fatalf("unexpected window size: %#v", got)
	}
}

func TestApplyDeviceDescriptorDefaultsScreen(t *testing.T) {
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{}
	desc := &playwright.DeviceDescriptor{
		UserAgent:         "ua-test",
		Viewport:          &playwright.Size{Width: 360, Height: 740},
		DeviceScaleFactor: 2,
		IsMobile:          true,
		HasTouch:          true,
	}
	applyDeviceDescriptor(&opts, desc)

	if opts.UserAgent == nil || *opts.UserAgent != "ua-test" {
		t.Fatalf("expected user agent to be set")
	}
	if opts.Viewport == nil || opts.Viewport.Width != 360 || opts.Viewport.Height != 740 {
		t.Fatalf("expected viewport to be set from device")
	}
	if opts.Screen == nil || opts.Screen.Width != 360 || opts.Screen.Height != 740 {
		t.Fatalf("expected screen to default to viewport")
	}
	if opts.DeviceScaleFactor == nil || *opts.DeviceScaleFactor != 2 {
		t.Fatalf("expected device scale factor to be set")
	}
	if opts.IsMobile == nil || !*opts.IsMobile {
		t.Fatalf("expected isMobile to be set")
	}
	if opts.HasTouch == nil || !*opts.HasTouch {
		t.Fatalf("expected hasTouch to be set")
	}
}

func TestApplyDeviceDescriptorUsesScreen(t *testing.T) {
	opts := playwright.BrowserTypeLaunchPersistentContextOptions{}
	desc := &playwright.DeviceDescriptor{
		Viewport: &playwright.Size{Width: 360, Height: 740},
		Screen:   &playwright.Size{Width: 720, Height: 1480},
	}
	applyDeviceDescriptor(&opts, desc)

	if opts.Screen == nil || opts.Screen.Width != 720 || opts.Screen.Height != 1480 {
		t.Fatalf("expected screen to use device screen")
	}
	if opts.Viewport == nil || opts.Viewport.Width != 360 || opts.Viewport.Height != 740 {
		t.Fatalf("expected viewport to use device viewport")
	}
}
