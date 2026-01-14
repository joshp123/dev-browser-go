package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

type globalOptions struct {
	profile     string
	headless    bool
	headed      bool
	output      string
	outPath     string
	windowSize  string
	windowScale float64
	window      *devbrowser.WindowSize
}

var globalOpts = &globalOptions{}

func bindGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&globalOpts.profile, "profile", getenvDefault("DEV_BROWSER_PROFILE", "default"), "Browser profile")
	cmd.PersistentFlags().BoolVar(&globalOpts.headless, "headless", defaultHeadless(), "Force headless")
	cmd.PersistentFlags().BoolVar(&globalOpts.headed, "headed", false, "Disable headless")
	cmd.PersistentFlags().StringVar(&globalOpts.windowSize, "window-size", "", "Viewport WxH")
	cmd.PersistentFlags().Float64Var(&globalOpts.windowScale, "window-scale", 1.0, "Viewport scale (1, 0.75, 0.5)")
	cmd.PersistentFlags().StringVar(&globalOpts.output, "output", "summary", "Output format (summary|json|path)")
	cmd.PersistentFlags().StringVar(&globalOpts.outPath, "out", "", "Output path when --output=path")
}

func applyGlobalOptions(cmd *cobra.Command) error {
	if err := resolveHeadless(cmd); err != nil {
		return err
	}
	if err := resolveWindow(cmd); err != nil {
		return err
	}
	if globalOpts.output != "summary" && globalOpts.output != "json" && globalOpts.output != "path" {
		return errors.New("--output must be summary|json|path")
	}
	return nil
}

func resolveHeadless(cmd *cobra.Command) error {
	headlessChanged := cmd.Flags().Changed("headless")
	headedChanged := cmd.Flags().Changed("headed")
	if headedChanged && headlessChanged {
		return errors.New("use either --headless or --headed")
	}
	if headedChanged {
		globalOpts.headless = false
		return nil
	}
	if headlessChanged && !globalOpts.headless {
		return errors.New("use --headed to disable headless")
	}
	return nil
}

func resolveWindow(cmd *cobra.Command) error {
	windowScaleChanged := cmd.Flags().Changed("window-scale")
	if strings.TrimSpace(globalOpts.windowSize) != "" && windowScaleChanged {
		return errors.New("use either --window-size or --window-scale")
	}
	scaleVal := 1.0
	if windowScaleChanged {
		scaleVal = globalOpts.windowScale
	}
	window, err := devbrowser.ResolveWindowSize(globalOpts.windowSize, scaleVal)
	if err != nil {
		return err
	}
	globalOpts.window = window
	return nil
}

func defaultHeadless() bool {
	if strings.TrimSpace(os.Getenv("HEADLESS")) == "" {
		return true
	}
	return envTruthy("HEADLESS")
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func getenvDefault(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func getenvInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func requireArgs(count int, errMsg string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) != count {
			return errors.New(errMsg)
		}
		return nil
	}
}

func maxArgs(max int, errMsg string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > max {
			return errors.New(errMsg)
		}
		return nil
	}
}
