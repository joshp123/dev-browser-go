package main

import (
	"github.com/spf13/cobra"
)

func newWaitCmd() *cobra.Command {
	var pageName string
	var strategy string
	var state string
	var timeout int
	var minWait int

	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for page state",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"strategy":    strategy,
				"state":       state,
				"timeout_ms":  timeout,
				"min_wait_ms": minWait,
			}
			return runWithPage(pageName, "wait", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&strategy, "strategy", "playwright", "Strategy")
	cmd.Flags().StringVar(&state, "state", "load", "State")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 10_000, "Timeout ms")
	cmd.Flags().IntVar(&minWait, "min-wait-ms", 0, "Min wait ms")

	return cmd
}
