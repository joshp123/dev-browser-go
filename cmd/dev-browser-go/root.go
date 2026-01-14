package main

import (
	"github.com/spf13/cobra"
)

const version = "0.2.0"

var rootCmd = &cobra.Command{
	Use:          "dev-browser-go",
	Short:        "ref-based browser automation (CLI + daemon)",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		return applyGlobalOptions(cmd)
	},
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	bindGlobalFlags(rootCmd)

	rootCmd.AddCommand(
		newDaemonCmd(),
		newStatusCmd(),
		newStartCmd(),
		newStopCmd(),
		newListPagesCmd(),
		newGotoCmd(),
		newSnapshotCmd(),
		newClickRefCmd(),
		newFillRefCmd(),
		newPressCmd(),
		newScreenshotCmd(),
		newBoundsCmd(),
		newConsoleCmd(),
		newSaveHTMLCmd(),
		newWaitCmd(),
		newCallCmd(),
		newActionsCmd(),
		newClosePageCmd(),
	)
}
