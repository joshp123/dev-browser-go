package main

import (
	"github.com/spf13/cobra"
)

func newFillRefCmd() *cobra.Command {
	var pageName string
	var timeout int

	cmd := &cobra.Command{
		Use:   "fill-ref <ref> <text>",
		Short: "Fill input by ref",
		Args:  requireArgs(2, "ref and text required"),
		RunE: func(_ *cobra.Command, args []string) error {
			payload := map[string]interface{}{
				"ref":        args[0],
				"text":       args[1],
				"timeout_ms": timeout,
			}
			return runWithPage(pageName, "fill_ref", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 15_000, "Timeout ms")

	return cmd
}
