package main

import (
	"encoding/json"
	"errors"

	"github.com/spf13/cobra"
)

func newCallCmd() *cobra.Command {
	var argsJSON string
	var pageName string

	cmd := &cobra.Command{
		Use:   "call <tool>",
		Short: "Call a tool by name",
		Args:  requireArgs(1, "tool name required"),
		RunE: func(_ *cobra.Command, args []string) error {
			argMap := map[string]interface{}{}
			if err := json.Unmarshal([]byte(argsJSON), &argMap); err != nil {
				return errors.New("invalid JSON for --args")
			}
			return runWithPage(pageName, args[0], argMap)
		},
	}

	cmd.Flags().StringVar(&argsJSON, "args", "{}", "JSON args for tool")
	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")

	return cmd
}
