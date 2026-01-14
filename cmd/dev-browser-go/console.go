package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
	"github.com/spf13/cobra"
)

func newConsoleCmd() *cobra.Command {
	var pageName string
	var since int64
	var limit int
	var levels []string

	cmd := &cobra.Command{
		Use:   "console",
		Short: "Read page console logs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if since < 0 {
				return fmt.Errorf("--since must be >= 0")
			}
			if limit < 0 {
				return fmt.Errorf("--limit must be >= 0")
			}
			base, err := startDaemonIfNeeded()
			if err != nil {
				return err
			}
			endpoint := fmt.Sprintf("%s/pages/%s/console", base, url.PathEscape(pageName))
			query := url.Values{}
			if cmd.Flags().Changed("limit") {
				query.Set("limit", strconv.Itoa(limit))
			}
			if len(levels) > 0 {
				query.Set("levels", strings.Join(levels, ","))
			}
			if since > 0 {
				query.Set("since", strconv.FormatInt(since, 10))
			}
			if encoded := query.Encode(); encoded != "" {
				endpoint += "?" + encoded
			}
			data, err := devbrowser.HTTPJSON("GET", endpoint, nil, 5*time.Second)
			if err != nil {
				return err
			}
			if ok, _ := data["ok"].(bool); !ok {
				return fmt.Errorf("console failed: %v", data["error"])
			}
			out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, data, globalOpts.outPath)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().Int64Var(&since, "since", 0, "Only return entries with id > since")
	cmd.Flags().IntVar(&limit, "limit", 200, "Max entries")
	cmd.Flags().StringArrayVar(&levels, "level", nil, "Log level (repeatable: debug,info,warning,error,all)")

	return cmd
}
