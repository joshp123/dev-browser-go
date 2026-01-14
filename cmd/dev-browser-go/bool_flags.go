package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type boolFlagRule struct {
	name  string
	hasNo bool
}

var boolFlagRules = []boolFlagRule{
	{name: "interactive-only", hasNo: true},
	{name: "include-headings", hasNo: true},
	{name: "full-page", hasNo: true},
	{name: "annotate-refs", hasNo: false},
}

func rejectBoolEqualsArgs(args []string) error {
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		if !strings.Contains(arg, "=") {
			continue
		}
		name := strings.TrimPrefix(arg, "--")
		if eq := strings.Index(name, "="); eq != -1 {
			name = name[:eq]
		}
		base := strings.TrimPrefix(name, "no-")
		for _, rule := range boolFlagRules {
			if rule.name != base {
				continue
			}
			if rule.hasNo {
				return fmt.Errorf("use --%s or --no-%s (omit =true/false)", rule.name, rule.name)
			}
			return fmt.Errorf("use --%s (omit =true/false)", rule.name)
		}
	}
	return nil
}

func applyNoFlag(cmd *cobra.Command, name string) error {
	noName := "no-" + name
	if cmd.Flags().Changed(noName) && cmd.Flags().Changed(name) {
		return fmt.Errorf("use --%s or --%s, not both", name, noName)
	}
	if !cmd.Flags().Changed(noName) {
		return nil
	}
	return cmd.Flags().Set(name, "false")
}
