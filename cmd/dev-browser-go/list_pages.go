package main

import (
	"fmt"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newListPagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-pages",
		Short: "List open pages",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := devbrowser.StartDaemon(globalOpts.profile, globalOpts.headless, globalOpts.window); err != nil {
				return err
			}
			base := devbrowser.DaemonBaseURL(globalOpts.profile)
			if base == "" {
				return fmt.Errorf("daemon state missing after start")
			}
			data, err := devbrowser.HTTPJSON("GET", base+"/pages", nil, 3*time.Second)
			if err != nil {
				return err
			}
			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, map[string]any{"pages": data["pages"]}, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}
}
