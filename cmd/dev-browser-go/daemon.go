package main

import (
	"log"
	"os"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

type daemonOptions struct {
	host      string
	port      int
	cdpPort   int
	stateFile string
}

func newDaemonCmd() *cobra.Command {
	opts := &daemonOptions{}
	cmd := &cobra.Command{
		Use:    "daemon",
		Short:  "Run daemon server",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDaemon(opts)
		},
	}
	cmd.Flags().StringVar(&opts.host, "host", getenvDefault("DEV_BROWSER_HOST", "127.0.0.1"), "Listen host")
	cmd.Flags().IntVar(&opts.port, "port", getenvInt("DEV_BROWSER_PORT", 0), "Listen port")
	cmd.Flags().IntVar(&opts.cdpPort, "cdp-port", getenvInt("DEV_BROWSER_CDP_PORT", 0), "CDP port")
	cmd.Flags().StringVar(&opts.stateFile, "state-file", getenvDefault("DEV_BROWSER_STATE_FILE", ""), "State file")
	return cmd
}

func runDaemon(opts *daemonOptions) error {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return devbrowser.ServeDaemon(devbrowser.DaemonOptions{
		Profile:   globalOpts.profile,
		Host:      opts.host,
		Port:      opts.port,
		CDPPort:   opts.cdpPort,
		Headless:  globalOpts.headless,
		Window:    globalOpts.window,
		StateFile: opts.stateFile,
		Logger:    logger,
	})
}
