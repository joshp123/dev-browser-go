package main

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
)

func runWithPage(pageName, tool string, args map[string]interface{}) error {
	ws, tid, err := devbrowser.EnsurePage(globalOpts.profile, globalOpts.headless, pageName, globalOpts.window)
	if err != nil {
		return err
	}
	pw, browser, page, err := devbrowser.OpenPage(ws, tid)
	if err != nil {
		return err
	}
	defer browser.Close()
	defer pw.Stop()

	res, err := devbrowser.RunCall(page, tool, args, devbrowser.ArtifactDir(globalOpts.profile))
	if err != nil {
		return err
	}
	out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, res, globalOpts.outPath)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func startDaemonIfNeeded() (string, error) {
	if err := devbrowser.StartDaemon(globalOpts.profile, globalOpts.headless, globalOpts.window); err != nil {
		return "", err
	}
	base := devbrowser.DaemonBaseURL(globalOpts.profile)
	if base == "" {
		return "", errors.New("daemon state missing after start")
	}
	return base, nil
}

func deletePage(name string) error {
	base, err := startDaemonIfNeeded()
	if err != nil {
		return err
	}
	encoded := url.PathEscape(name)
	data, err := devbrowser.HTTPJSON("DELETE", base+"/pages/"+encoded, nil, 5*time.Second)
	if err != nil {
		return err
	}
	if ok, _ := data["ok"].(bool); !ok {
		return fmt.Errorf("close failed: %v", data["error"])
	}
	out, err := devbrowser.WriteOutput(globalOpts.profile, globalOpts.output, map[string]any{"page": name, "closed": true}, globalOpts.outPath)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
