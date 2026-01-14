package devbrowser

import (
	"fmt"
	"sort"
	"strings"

	"github.com/playwright-community/playwright-go"
)

func resolveDeviceDescriptor(pw *playwright.Playwright, name string) (*playwright.DeviceDescriptor, error) {
	deviceName := strings.TrimSpace(name)
	if deviceName == "" {
		return nil, nil
	}
	if pw == nil {
		return nil, fmt.Errorf("playwright not initialized for device lookup")
	}
	desc, ok := pw.Devices[deviceName]
	if !ok || desc == nil {
		return nil, fmt.Errorf("unknown device %q", deviceName)
	}
	return desc, nil
}

func deviceWindowSize(desc *playwright.DeviceDescriptor) *WindowSize {
	if desc == nil || desc.Viewport == nil {
		return nil
	}
	return &WindowSize{Width: desc.Viewport.Width, Height: desc.Viewport.Height}
}

func applyDeviceDescriptor(opts *playwright.BrowserTypeLaunchPersistentContextOptions, desc *playwright.DeviceDescriptor) {
	if opts == nil || desc == nil {
		return
	}
	if desc.UserAgent != "" {
		opts.UserAgent = playwright.String(desc.UserAgent)
	}
	if desc.Viewport != nil {
		opts.Viewport = &playwright.Size{Width: desc.Viewport.Width, Height: desc.Viewport.Height}
	}
	if desc.Screen != nil {
		opts.Screen = &playwright.Size{Width: desc.Screen.Width, Height: desc.Screen.Height}
	} else if desc.Viewport != nil {
		opts.Screen = &playwright.Size{Width: desc.Viewport.Width, Height: desc.Viewport.Height}
	}
	if desc.DeviceScaleFactor > 0 {
		opts.DeviceScaleFactor = playwright.Float(desc.DeviceScaleFactor)
	}
	opts.IsMobile = playwright.Bool(desc.IsMobile)
	opts.HasTouch = playwright.Bool(desc.HasTouch)
}

func ListDeviceNames() ([]string, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}
	defer pw.Stop()

	names := make([]string, 0, len(pw.Devices))
	for name := range pw.Devices {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}
