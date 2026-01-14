package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newBoundsCmd() *cobra.Command {
	var pageName string
	var selector string
	var ariaRole string
	var ariaName string
	var nth int
	var timeout int

	cmd := &cobra.Command{
		Use:   "bounds [selector]",
		Short: "Get element bounding box",
		Args:  maxArgs(1, "too many arguments"),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 && strings.TrimSpace(selector) == "" {
				selector = args[0]
			}
			if strings.TrimSpace(selector) == "" && strings.TrimSpace(ariaRole) == "" {
				return errors.New("selector or --aria-role required")
			}
			payload := map[string]interface{}{
				"nth":        nth,
				"timeout_ms": timeout,
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
			return runWithPage(pageName, "bounds", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector")
	cmd.Flags().StringVar(&ariaRole, "aria-role", "", "ARIA role")
	cmd.Flags().StringVar(&ariaName, "aria-name", "", "ARIA name")
	cmd.Flags().IntVar(&nth, "nth", 1, "Nth match (1-based)")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 5_000, "Timeout ms")

	return cmd
}
