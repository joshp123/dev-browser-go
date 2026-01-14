package main

import (
	"github.com/spf13/cobra"
)

func newClosePageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-page <name>",
		Short: "Close named page",
		Args:  requireArgs(1, "page name required"),
		RunE: func(_ *cobra.Command, args []string) error {
			return deletePage(args[0])
		},
	}

	return cmd
}
