package main

import (
	"github.com/spf13/cobra"
)

func newPressCmd() *cobra.Command {
	var pageName string

	cmd := &cobra.Command{
		Use:   "press <key>",
		Short: "Send key press",
		Args:  requireArgs(1, "key required"),
		RunE: func(_ *cobra.Command, args []string) error {
			payload := map[string]interface{}{"key": args[0]}
			return runWithPage(pageName, "press", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")

	return cmd
}
