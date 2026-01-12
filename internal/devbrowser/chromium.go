package devbrowser

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type WindowSize struct {
	Width  int
	Height int
}

var (
	defaultWindowSize = WindowSize{Width: 7680, Height: 2160}
	windowSizeRe      = regexp.MustCompile(`^\s*(\d+)\s*[xX*,]\s*(\d+)\s*$`)
)

func DefaultWindowSize() WindowSize {
	return defaultWindowSize
}

func ParseWindowSize(raw string) (*WindowSize, error) {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), "px", ""))
	if normalized == "" {
		return nil, nil
	}
	match := windowSizeRe.FindStringSubmatch(normalized)
	if len(match) != 3 {
		return nil, fmt.Errorf("window size must be WIDTHxHEIGHT (e.g. 7680x2160)")
	}
	w, _ := strconv.Atoi(match[1])
	h, _ := strconv.Atoi(match[2])
	if w < 1 || h < 1 {
		return nil, fmt.Errorf("window size must be positive (e.g. 7680x2160)")
	}
	return &WindowSize{Width: w, Height: h}, nil
}

func ResolveWindowSize(raw string, scale float64) (*WindowSize, error) {
	if raw != "" && scale > 0 && scale != 1 {
		return nil, fmt.Errorf("use either --window-size or --window-scale, not both")
	}

	if raw != "" {
		return ParseWindowSize(raw)
	}

	base := defaultWindowSize
	if scale <= 0 {
		scale = 1
	}
	width := int(math.Round(float64(base.Width) * scale))
	height := int(math.Round(float64(base.Height) * scale))
	if width < 1 || height < 1 {
		return nil, fmt.Errorf("resolved window size is invalid")
	}
	return &WindowSize{Width: width, Height: height}, nil
}

func ChromiumLaunchArgs(cdpPort int, window *WindowSize) []string {
	args := []string{}
	if cdpPort > 0 {
		args = append(args, fmt.Sprintf("--remote-debugging-port=%d", cdpPort))
	}
	if !envTruthy("DEV_BROWSER_USE_KEYCHAIN") {
		args = append(args, "--use-mock-keychain")
	}
	if window != nil {
		args = append(args, fmt.Sprintf("--window-size=%d,%d", window.Width, window.Height))
	}
	return args
}
