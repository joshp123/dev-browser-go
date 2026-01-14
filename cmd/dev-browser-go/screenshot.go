package main

import (
	"strings"

	"github.com/spf13/cobra"
)

func newScreenshotCmd() *cobra.Command {
	var pageName string
	var pathArg string
	var fullPage bool
	var annotate bool
	var crop string
	var selector string
	var ariaRole string
	var ariaName string
	var nth int
	var padding int
	var timeout int

	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Save screenshot",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return applyNoFlag(cmd, "full-page")
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"full_page":     fullPage,
				"annotate_refs": annotate,
				"nth":           nth,
				"padding_px":    padding,
				"timeout_ms":    timeout,
			}
			if strings.TrimSpace(pathArg) != "" {
				payload["path"] = pathArg
			}
			if strings.TrimSpace(crop) != "" {
				payload["crop"] = crop
			}
			if strings.TrimSpace(selector) != "" {
				payload["selector"] = selector
			}
			if strings.TrimSpace(ariaRole) != "" {
				payload["aria_role"] = ariaRole
			}
			if strings.TrimSpace(ariaName) != "" {
				payload["aria_name"] = ariaName
			}
			return runWithPage(pageName, "screenshot", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&pathArg, "path", "", "Output path")
	cmd.Flags().BoolVar(&fullPage, "full-page", true, "Full page")
	cmd.Flags().BoolVar(&annotate, "annotate-refs", false, "Annotate refs")
	cmd.Flags().StringVar(&crop, "crop", "", "Crop x,y,w,h")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector for element crop")
	cmd.Flags().StringVar(&ariaRole, "aria-role", "", "ARIA role for element crop")
	cmd.Flags().StringVar(&ariaName, "aria-name", "", "ARIA name for element crop")
	cmd.Flags().IntVar(&nth, "nth", 1, "Nth match (1-based)")
	cmd.Flags().IntVar(&padding, "padding-px", 10, "Padding around element in px")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 5_000, "Timeout ms for element wait")
	cmd.Flags().Bool("no-full-page", false, "Disable full page")

	return cmd
}
