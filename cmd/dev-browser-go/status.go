package main

import (
	"fmt"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(_ *cobra.Command, _ []string) error {
			if devbrowser.IsDaemonHealthy(globalOpts.profile) {
				fmt.Printf("ok profile=%s url=%s\n", globalOpts.profile, devbrowser.DaemonBaseURL(globalOpts.profile))
				return nil
			}
			fmt.Printf("not running profile=%s\n", globalOpts.profile)
			return nil
		},
	}
}
