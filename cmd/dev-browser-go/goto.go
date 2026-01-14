package main

import (
	"github.com/spf13/cobra"
)

func newGotoCmd() *cobra.Command {
	var pageName string
	var waitUntil string
	var timeout int

	cmd := &cobra.Command{
		Use:   "goto <url>",
		Short: "Navigate to URL",
		Args:  requireArgs(1, "url required"),
		RunE: func(_ *cobra.Command, args []string) error {
			payload := map[string]interface{}{
				"url":        args[0],
				"wait_until": waitUntil,
				"timeout_ms": timeout,
			}
			return runWithPage(pageName, "goto", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&waitUntil, "wait-until", "domcontentloaded", "Wait strategy")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 45_000, "Timeout ms")

	return cmd
}
