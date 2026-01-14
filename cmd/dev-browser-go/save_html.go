package main

import (
	"github.com/spf13/cobra"
)

func newSaveHTMLCmd() *cobra.Command {
	var pageName string
	var pathArg string

	cmd := &cobra.Command{
		Use:   "save-html",
		Short: "Save page HTML",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{"path": pathArg}
			return runWithPage(pageName, "save_html", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&pathArg, "path", "", "Output path")

	return cmd
}
