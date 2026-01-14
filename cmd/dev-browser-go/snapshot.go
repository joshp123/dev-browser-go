package main

import (
	"errors"

	"github.com/spf13/cobra"
)

func newSnapshotCmd() *cobra.Command {
	var pageName string
	var engine string
	var format string
	var interactiveOnly bool
	var includeHeadings bool
	var maxItems int
	var maxChars int

	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Get accessibility snapshot",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"engine":           engine,
				"format":           format,
				"interactive_only": interactiveOnly,
				"include_headings": includeHeadings,
				"max_items":        maxItems,
				"max_chars":        maxChars,
			}
			return runWithPage(pageName, "snapshot", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&engine, "engine", "simple", "Engine (simple|aria)")
	cmd.Flags().StringVar(&format, "format", "list", "Format (list|json|yaml)")
	cmd.Flags().BoolVar(&interactiveOnly, "interactive-only", true, "Only interactive elements")
	cmd.Flags().BoolVar(&includeHeadings, "include-headings", true, "Include headings")
	cmd.Flags().IntVar(&maxItems, "max-items", 80, "Max items")
	cmd.Flags().IntVar(&maxChars, "max-chars", 8000, "Max chars")

	cmd.Flags().Bool("no-interactive-only", false, "Include non-interactive elements")
	cmd.Flags().Bool("no-include-headings", false, "Exclude headings")

	cmd.PreRunE = func(cmd *cobra.Command, _ []string) error {
		if err := applyNoFlag(cmd, "interactive-only"); err != nil {
			return err
		}
		if err := applyNoFlag(cmd, "include-headings"); err != nil {
			return err
		}
		if maxItems < 0 {
			return errors.New("--max-items must be >= 0")
		}
		if maxChars < 0 {
			return errors.New("--max-chars must be >= 0")
		}
		return nil
	}

	return cmd
}
