package main

import (
	"testing"
)

func TestParseGlobalsDeviceFlag(t *testing.T) {
	g, rest, err := parseGlobals([]string{"--device", "Pixel 5", "status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.device != "Pixel 5" {
		t.Fatalf("expected device to be set, got %q", g.device)
	}
	if g.window != nil {
		t.Fatalf("expected window to be nil with device")
	}
	if len(rest) != 1 || rest[0] != "status" {
		t.Fatalf("unexpected remaining args: %#v", rest)
	}
}

func TestParseGlobalsDeviceConflictsWithWindow(t *testing.T) {
	_, _, err := parseGlobals([]string{"--device", "Pixel 5", "--window-size", "100x200", "status"})
	if err == nil {
		t.Fatalf("expected error for device + window-size")
	}
}

func TestParseGlobalsUsesEnvWindowSize(t *testing.T) {
	t.Setenv("DEV_BROWSER_WINDOW_SIZE", "320x640")
	g, rest, err := parseGlobals([]string{"status"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.window == nil || g.window.Width != 320 || g.window.Height != 640 {
		t.Fatalf("expected window from env, got %#v", g.window)
	}
	if len(rest) != 1 || rest[0] != "status" {
		t.Fatalf("unexpected remaining args: %#v", rest)
	}
}
