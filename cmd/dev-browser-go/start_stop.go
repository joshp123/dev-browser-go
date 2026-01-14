package main

import (
	"fmt"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := devbrowser.StartDaemon(globalOpts.profile, globalOpts.headless, globalOpts.window); err != nil {
				return err
			}
			fmt.Printf("started profile=%s url=%s\n", globalOpts.profile, devbrowser.DaemonBaseURL(globalOpts.profile))
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			stopped, err := devbrowser.StopDaemon(globalOpts.profile)
			if err != nil {
				return err
			}
			if stopped {
				fmt.Printf("stopped profile=%s\n", globalOpts.profile)
				return nil
			}
			fmt.Printf("not running profile=%s\n", globalOpts.profile)
			return nil
		},
	}
}
