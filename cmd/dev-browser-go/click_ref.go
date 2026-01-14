package main

import (
	"github.com/spf13/cobra"
)

func newClickRefCmd() *cobra.Command {
	var pageName string
	var timeout int

	cmd := &cobra.Command{
		Use:   "click-ref <ref>",
		Short: "Click element by ref",
		Args:  requireArgs(1, "ref required"),
		RunE: func(_ *cobra.Command, args []string) error {
			payload := map[string]interface{}{
				"ref":        args[0],
				"timeout_ms": timeout,
			}
			return runWithPage(pageName, "click_ref", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 15_000, "Timeout ms")

	return cmd
}
