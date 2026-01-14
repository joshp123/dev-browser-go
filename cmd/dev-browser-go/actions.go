package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newActionsCmd() *cobra.Command {
	var callsArg string
	var pageName string

	cmd := &cobra.Command{
		Use:   "actions",
		Short: "Batch tool calls from JSON",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			raw := strings.TrimSpace(callsArg)
			if raw == "" {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				raw = string(b)
			}
			var calls []map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &calls); err != nil {
				return errors.New("invalid JSON for --calls/stdin")
			}
			ws, tid, err := devbrowser.EnsurePage(globalOpts.profile, globalOpts.headless, pageName, globalOpts.window)
			if err != nil {
				return err
			}
			pw, browser, page, err := devbrowser.OpenPage(ws, tid)
			if err != nil {
				return err
			}
			defer browser.Close()
			defer pw.Stop()

			res, err := devbrowser.RunActions(page, calls, devbrowser.ArtifactDir(globalOpts.profile))
			if err != nil {
				return err
			}
			output := map[string]any{"results": res.Results}
			if res.Snapshot != "" {
				output["snapshot"] = res.Snapshot
			}
			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, output, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&callsArg, "calls", "", "JSON calls array (or stdin)")
	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")

	return cmd
}
