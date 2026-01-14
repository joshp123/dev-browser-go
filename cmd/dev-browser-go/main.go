package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	devbrowser "github.com/joshp123/dev-browser-go/internal/devbrowser"
)

const version = "0.2.0"

type globals struct {
	profile  string
	headless bool
	output   string
	outPath  string
	window   *devbrowser.WindowSize
}

type repeatableStringFlag struct {
	values []string
}

func (flagValue *repeatableStringFlag) String() string {
	if flagValue == nil {
		return ""
	}
	return strings.Join(flagValue.values, ",")
}

func (flagValue *repeatableStringFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("level must be non-empty")
	}
	flagValue.values = append(flagValue.values, trimmed)
	return nil
}

type optionalIntFlag struct {
	value int
	set   bool
}

func (flagValue *optionalIntFlag) String() string {
	if flagValue == nil || !flagValue.set {
		return ""
	}
	return strconv.Itoa(flagValue.value)
}

func (flagValue *optionalIntFlag) Set(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if parsed < 0 {
		return errors.New("limit must be >= 0")
	}
	flagValue.value = parsed
	flagValue.set = true
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	if args[0] == "--help" || args[0] == "-h" {
		printUsage()
		return nil
	}
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Println(version)
		return nil
	}

	if args[0] == "--daemon" {
		if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
			printDaemonUsage()
			return nil
		}
		return runDaemon(args[1:])
	}

	g, rest, err := parseGlobals(args)
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		printUsage()
		return nil
	}

	cmd := rest[0]
	rest = rest[1:]

	if len(rest) > 0 {
		if rest[0] == "--help" || rest[0] == "-h" {
			printCommandUsage(cmd)
			return nil
		}
	}

	switch cmd {
	case "status":
		if devbrowser.IsDaemonHealthy(g.profile) {
			fmt.Printf("ok profile=%s url=%s\n", g.profile, devbrowser.DaemonBaseURL(g.profile))
			return nil
		}
		fmt.Printf("not running profile=%s\n", g.profile)
		return nil

	case "start":
		if err := devbrowser.StartDaemon(g.profile, g.headless, g.window); err != nil {
			return err
		}
		fmt.Printf("started profile=%s url=%s\n", g.profile, devbrowser.DaemonBaseURL(g.profile))
		return nil

	case "stop":
		stopped, err := devbrowser.StopDaemon(g.profile)
		if err != nil {
			return err
		}
		if stopped {
			fmt.Printf("stopped profile=%s\n", g.profile)
			return nil
		}
		fmt.Printf("not running profile=%s\n", g.profile)
		return nil

	case "list-pages":
		if err := devbrowser.StartDaemon(g.profile, g.headless, g.window); err != nil {
			return err
		}
		base := devbrowser.DaemonBaseURL(g.profile)
		if base == "" {
			return errors.New("daemon state missing after start")
		}
		data, err := devbrowser.HTTPJSON("GET", base+"/pages", nil, 3*time.Second)
		if err != nil {
			return err
		}
		out, err := devbrowser.WriteOutput(g.profile, g.output, map[string]any{"pages": data["pages"]}, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil

	case "call":
		fs := flag.NewFlagSet("call", flag.ContinueOnError)
		argsJSON := fs.String("args", "{}", "JSON args for tool")
		page := fs.String("page", "main", "Page name")
		fs.Usage = func() { printCommandUsage("call") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("tool name required")
		}
		tool := fs.Arg(0)
		argMap := map[string]interface{}{}
		if err := json.Unmarshal([]byte(*argsJSON), &argMap); err != nil {
			return errors.New("invalid JSON for --args")
		}
		return runWithPage(g, *page, tool, argMap)

	case "goto":
		fs := flag.NewFlagSet("goto", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		waitUntil := fs.String("wait-until", "domcontentloaded", "Wait strategy")
		timeout := fs.Int("timeout-ms", 45_000, "Timeout ms")
		fs.Usage = func() { printCommandUsage("goto") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("url required")
		}
		urlVal := fs.Arg(0)
		return runWithPage(g, *pageName, "goto", map[string]interface{}{"url": urlVal, "wait_until": *waitUntil, "timeout_ms": *timeout})

	case "snapshot":
		fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		engine := fs.String("engine", "simple", "Engine (simple|aria)")
		format := fs.String("format", "list", "Format (list|json|yaml)")
		interactiveOnly := fs.Bool("interactive-only", true, "Only interactive elements")
		includeHeadings := fs.Bool("include-headings", true, "Include headings")
		maxItems := fs.Int("max-items", 80, "Max items")
		maxChars := fs.Int("max-chars", 8000, "Max chars")
		fs.Usage = func() { printCommandUsage("snapshot") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		return runWithPage(g, *pageName, "snapshot", map[string]interface{}{
			"engine":           *engine,
			"format":           *format,
			"interactive_only": *interactiveOnly,
			"include_headings": *includeHeadings,
			"max_items":        *maxItems,
			"max_chars":        *maxChars,
		})

	case "console":
		fs := flag.NewFlagSet("console", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		since := fs.Int64("since", 0, "Only return entries with id > since")
		limit := optionalIntFlag{}
		levels := repeatableStringFlag{}
		fs.Var(&limit, "limit", "Max entries")
		fs.Var(&levels, "level", "Log level (repeatable: debug,info,warning,error,all)")
		fs.Usage = func() { printCommandUsage("console") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if *since < 0 {
			return errors.New("--since must be >= 0")
		}
		if err := devbrowser.StartDaemon(g.profile, g.headless, g.window); err != nil {
			return err
		}
		base := devbrowser.DaemonBaseURL(g.profile)
		if base == "" {
			return errors.New("daemon state missing after start")
		}
		endpoint := fmt.Sprintf("%s/pages/%s/console", base, url.PathEscape(*pageName))
		query := url.Values{}
		if limit.set {
			query.Set("limit", strconv.Itoa(limit.value))
		}
		if len(levels.values) > 0 {
			query.Set("levels", strings.Join(levels.values, ","))
		}
		if *since > 0 {
			query.Set("since", strconv.FormatInt(*since, 10))
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
		out, err := devbrowser.WriteOutput(g.profile, g.output, data, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil

	case "click-ref":
		fs := flag.NewFlagSet("click-ref", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		timeout := fs.Int("timeout-ms", 15_000, "Timeout ms")
		fs.Usage = func() { printCommandUsage("click-ref") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("ref required")
		}
		ref := fs.Arg(0)
		return runWithPage(g, *pageName, "click_ref", map[string]interface{}{"ref": ref, "timeout_ms": *timeout})

	case "fill-ref":
		fs := flag.NewFlagSet("fill-ref", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		timeout := fs.Int("timeout-ms", 15_000, "Timeout ms")
		fs.Usage = func() { printCommandUsage("fill-ref") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() < 2 {
			return errors.New("ref and text required")
		}
		ref := fs.Arg(0)
		text := fs.Arg(1)
		return runWithPage(g, *pageName, "fill_ref", map[string]interface{}{"ref": ref, "text": text, "timeout_ms": *timeout})

	case "press":
		fs := flag.NewFlagSet("press", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		fs.Usage = func() { printCommandUsage("press") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("key required")
		}
		key := fs.Arg(0)
		return runWithPage(g, *pageName, "press", map[string]interface{}{"key": key})

	case "screenshot":
		fs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		pathArg := fs.String("path", "", "Output path")
		fullPage := fs.Bool("full-page", true, "Full page")
		annotate := fs.Bool("annotate-refs", false, "Annotate refs")
		crop := fs.String("crop", "", "Crop x,y,w,h")
		selector := fs.String("selector", "", "CSS selector for element crop")
		ariaRole := fs.String("aria-role", "", "ARIA role for element crop")
		ariaName := fs.String("aria-name", "", "ARIA name for element crop")
		nth := fs.Int("nth", 1, "Nth match (1-based)")
		padding := fs.Int("padding-px", 10, "Padding around element in px")
		timeout := fs.Int("timeout-ms", 5_000, "Timeout ms for element wait")
		fs.Usage = func() { printCommandUsage("screenshot") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		payload := map[string]interface{}{"full_page": *fullPage, "annotate_refs": *annotate, "nth": *nth, "padding_px": *padding, "timeout_ms": *timeout}
		if strings.TrimSpace(*pathArg) != "" {
			payload["path"] = *pathArg
		}
		if strings.TrimSpace(*crop) != "" {
			payload["crop"] = *crop
		}
		if strings.TrimSpace(*selector) != "" {
			payload["selector"] = *selector
		}
		if strings.TrimSpace(*ariaRole) != "" {
			payload["aria_role"] = *ariaRole
		}
		if strings.TrimSpace(*ariaName) != "" {
			payload["aria_name"] = *ariaName
		}
		return runWithPage(g, *pageName, "screenshot", payload)

	case "bounds":
		fs := flag.NewFlagSet("bounds", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		selector := fs.String("selector", "", "CSS selector")
		ariaRole := fs.String("aria-role", "", "ARIA role")
		ariaName := fs.String("aria-name", "", "ARIA name")
		nth := fs.Int("nth", 1, "Nth match (1-based)")
		timeout := fs.Int("timeout-ms", 5_000, "Timeout ms")
		fs.Usage = func() { printCommandUsage("bounds") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() > 0 && strings.TrimSpace(*selector) == "" {
			*selector = fs.Arg(0)
		}
		if strings.TrimSpace(*selector) == "" && strings.TrimSpace(*ariaRole) == "" {
			return errors.New("selector or --aria-role required")
		}
		payload := map[string]interface{}{"nth": *nth, "timeout_ms": *timeout}
		if strings.TrimSpace(*selector) != "" {
			payload["selector"] = *selector
		}
		if strings.TrimSpace(*ariaRole) != "" {
			payload["aria_role"] = *ariaRole
		}
		if strings.TrimSpace(*ariaName) != "" {
			payload["aria_name"] = *ariaName
		}
		return runWithPage(g, *pageName, "bounds", payload)

	case "save-html":
		fs := flag.NewFlagSet("save-html", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		pathArg := fs.String("path", "", "Output path")
		fs.Usage = func() { printCommandUsage("save-html") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		return runWithPage(g, *pageName, "save_html", map[string]interface{}{"path": *pathArg})

	case "actions":
		fs := flag.NewFlagSet("actions", flag.ContinueOnError)
		callsArg := fs.String("calls", "", "JSON calls array (or stdin)")
		pageName := fs.String("page", "main", "Page name")
		fs.Usage = func() { printCommandUsage("actions") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		raw := strings.TrimSpace(*callsArg)
		if raw == "" {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			raw = string(b)
		}
		var calls []map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &calls); err != nil {
			return errors.New("invalid JSON for --calls/stdin")
		}
		ws, tid, err := devbrowser.EnsurePage(g.profile, g.headless, *pageName, g.window)
		if err != nil {
			return err
		}
		pw, browser, page, err := devbrowser.OpenPage(ws, tid)
		if err != nil {
			return err
		}
		defer browser.Close()
		defer pw.Stop()

		res, err := devbrowser.RunActions(page, calls, devbrowser.ArtifactDir(g.profile))
		if err != nil {
			return err
		}
		output := map[string]any{"results": res.Results}
		if res.Snapshot != "" {
			output["snapshot"] = res.Snapshot
		}
		out, err := devbrowser.WriteOutput(g.profile, g.output, output, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil

	case "wait":
		fs := flag.NewFlagSet("wait", flag.ContinueOnError)
		pageName := fs.String("page", "main", "Page name")
		strategy := fs.String("strategy", "playwright", "Strategy")
		stateVal := fs.String("state", "load", "State")
		timeout := fs.Int("timeout-ms", 10_000, "Timeout ms")
		minWait := fs.Int("min-wait-ms", 0, "Min wait ms")
		fs.Usage = func() { printCommandUsage("wait") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		return runWithPage(g, *pageName, "wait", map[string]interface{}{"strategy": *strategy, "state": *stateVal, "timeout_ms": *timeout, "min_wait_ms": *minWait})

	case "close-page":
		fs := flag.NewFlagSet("close-page", flag.ContinueOnError)
		fs.Usage = func() { printCommandUsage("close-page") }
		if err := fs.Parse(rest); err != nil {
			if err == flag.ErrHelp {
				return nil
			}
			return err
		}
		if fs.NArg() < 1 {
			return errors.New("page name required")
		}
		name := fs.Arg(0)
		if err := devbrowser.StartDaemon(g.profile, g.headless, g.window); err != nil {
			return err
		}
		base := devbrowser.DaemonBaseURL(g.profile)
		if base == "" {
			return errors.New("daemon state missing after start")
		}
		encoded := url.PathEscape(name)
		data, err := devbrowser.HTTPJSON("DELETE", base+"/pages/"+encoded, nil, 5*time.Second)
		if err != nil {
			return err
		}
		if ok, _ := data["ok"].(bool); !ok {
			return fmt.Errorf("close failed: %v", data["error"])
		}
		out, err := devbrowser.WriteOutput(g.profile, g.output, map[string]any{"page": name, "closed": true}, g.outPath)
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	}

	printUsage()
	return fmt.Errorf("unknown command: %s", cmd)
}

func runWithPage(g globals, pageName string, tool string, args map[string]interface{}) error {
	ws, tid, err := devbrowser.EnsurePage(g.profile, g.headless, pageName, g.window)
	if err != nil {
		return err
	}
	pw, browser, page, err := devbrowser.OpenPage(ws, tid)
	if err != nil {
		return err
	}
	defer browser.Close()
	defer pw.Stop()

	res, err := devbrowser.RunCall(page, tool, args, devbrowser.ArtifactDir(g.profile))
	if err != nil {
		return err
	}
	out, err := devbrowser.WriteOutput(g.profile, g.output, res, g.outPath)
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}

func runDaemon(args []string) error {
	profile := getenvDefault("DEV_BROWSER_PROFILE", "default")
	host := getenvDefault("DEV_BROWSER_HOST", "127.0.0.1")
	port := getenvInt("DEV_BROWSER_PORT", 0)
	cdpPort := getenvInt("DEV_BROWSER_CDP_PORT", 0)
	headless := true
	if raw := strings.TrimSpace(os.Getenv("HEADLESS")); raw != "" {
		headless = envTruthy("HEADLESS")
	}
	stateFile := getenvDefault("DEV_BROWSER_STATE_FILE", "")
	windowSizeRaw := ""
	windowScale := 1.0
	windowScaleSet := false

	fs := flag.NewFlagSet("dev-browser-go-daemon", flag.ContinueOnError)
	fs.StringVar(&profile, "profile", profile, "Profile name")
	fs.StringVar(&host, "host", host, "Listen host")
	fs.IntVar(&port, "port", port, "Listen port")
	fs.IntVar(&cdpPort, "cdp-port", cdpPort, "CDP port")
	fs.BoolVar(&headless, "headless", headless, "Headless")
	fs.StringVar(&stateFile, "state-file", stateFile, "State file")
	fs.StringVar(&windowSizeRaw, "window-size", windowSizeRaw, "Viewport WxH")
	fs.Float64Var(&windowScale, "window-scale", windowScale, "Viewport scale (e.g. 1, 0.75, 0.5)")
	fs.Usage = func() { printDaemonUsage() }
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}

	if fs.Lookup("window-scale").Value.String() != "1" {
		windowScaleSet = true
	}
	if windowSizeRaw != "" && windowScaleSet {
		return fmt.Errorf("use either --window-size or --window-scale")
	}

	scaleVal := windowScale
	if !windowScaleSet {
		scaleVal = 1
	}
	window, err := devbrowser.ResolveWindowSize(windowSizeRaw, scaleVal)
	if err != nil {
		return err
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	return devbrowser.ServeDaemon(devbrowser.DaemonOptions{Profile: profile, Host: host, Port: port, CDPPort: cdpPort, Headless: headless, Window: window, StateFile: stateFile, Logger: logger})
}

func parseGlobals(args []string) (globals, []string, error) {
	g := globals{
		profile:  getenvDefault("DEV_BROWSER_PROFILE", "default"),
		headless: true,
		output:   "summary",
		outPath:  "",
	}

	if raw := strings.TrimSpace(os.Getenv("HEADLESS")); raw != "" {
		g.headless = envTruthy("HEADLESS")
	}

	remaining := []string{}
	i := 0
	windowSizeRaw := ""
	windowScale := 1.0
	windowScaleSet := false
	for i < len(args) {
		a := args[i]
		switch a {
		case "--profile":
			if i+1 >= len(args) {
				return g, nil, errors.New("--profile requires value")
			}
			g.profile = args[i+1]
			i += 2
		case "--headless":
			g.headless = true
			i++
		case "--headed":
			g.headless = false
			i++
		case "--output":
			if i+1 >= len(args) {
				return g, nil, errors.New("--output requires value")
			}
			g.output = args[i+1]
			i += 2
		case "--out":
			if i+1 >= len(args) {
				return g, nil, errors.New("--out requires value")
			}
			g.outPath = args[i+1]
			i += 2
		case "--window-size":
			if i+1 >= len(args) {
				return g, nil, errors.New("--window-size requires value")
			}
			windowSizeRaw = args[i+1]
			i += 2
		case "--window-scale":
			if i+1 >= len(args) {
				return g, nil, errors.New("--window-scale requires value")
			}
			scaleStr := args[i+1]
			val, err := strconv.ParseFloat(scaleStr, 64)
			if err != nil {
				return g, nil, fmt.Errorf("--window-scale must be a number (e.g. 1, 0.75, 0.5)")
			}
			windowScale = val
			windowScaleSet = true
			i += 2
		default:
			remaining = args[i:]
			i = len(args)
		}
	}

	if g.output != "summary" && g.output != "json" && g.output != "path" {
		return g, nil, errors.New("--output must be summary|json|path")
	}

	scaleVal := windowScale
	if !windowScaleSet {
		scaleVal = 1
	}
	window, err := devbrowser.ResolveWindowSize(windowSizeRaw, scaleVal)
	if err != nil {
		return g, nil, err
	}
	g.window = window

	return g, remaining, nil
}

func getenvDefault(name, def string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	return v
}

func getenvInt(name string, def int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envTruthy(name string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func printUsage() {
	fmt.Fprintf(os.Stdout, `dev-browser-go %s - ref-based browser automation (CLI + daemon)

Usage:
  dev-browser-go [global flags] <command> [args]
  dev-browser-go --daemon [daemon flags]

Global flags:
  --profile <name>           Browser profile (default env DEV_BROWSER_PROFILE or "default")
  --headless                 Force headless
  --headed                   Disable headless
  --window-size WxH          Viewport size (default 7680x2160 ultrawide)
  --window-scale SCALE       Viewport scale (1, 0.75, 0.5)
  --output summary|json|path Output format (default: summary)
  --out <path>               Output path when --output=path
  --help, -h                 Show help
  --version, -v              Show version

Commands:
  goto <url> [--page name] [--wait-until state] [--timeout-ms ms]
  snapshot [--page name] [--engine simple|aria] [--format list|json|yaml] [--interactive-only] [--include-headings] [--max-items N] [--max-chars N]
  click-ref <ref> [--page name] [--timeout-ms ms]
  fill-ref <ref> <text> [--page name] [--timeout-ms ms]
  press <key> [--page name]
  screenshot [--page name] [--path PATH] [--full-page] [--annotate-refs] [--crop x,y,w,h] [--selector CSS] [--aria-role ROLE] [--aria-name NAME] [--nth N] [--padding-px PX] [--timeout-ms MS]
  bounds [selector] [--page name] [--aria-role ROLE] [--aria-name NAME] [--nth N] [--timeout-ms MS]
  console [--page name] [--since id] [--limit N] [--level lvl]
  save-html [--page name] [--path PATH]
  call <tool> [--args JSON] [--page name]
  actions [--calls JSON] [--page name] (reads stdin if empty)
  wait [--page name] [--strategy] [--state] [--timeout-ms] [--min-wait-ms]
  list-pages
  close-page <name>
  status | start | stop

Run "dev-browser-go <command> --help" for command details.
`, version)
}

func printCommandUsage(cmd string) {
	switch cmd {
	case "goto":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] goto <url> [--page name] [--wait-until state] [--timeout-ms ms]\n")
	case "snapshot":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] snapshot [--page name] [--engine simple|aria] [--format list|json|yaml] [--interactive-only] [--include-headings] [--max-items N] [--max-chars N]\n")
	case "click-ref":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] click-ref <ref> [--page name] [--timeout-ms ms]\n")
	case "fill-ref":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] fill-ref <ref> <text> [--page name] [--timeout-ms ms]\n")
	case "press":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] press <key> [--page name]\n")
	case "screenshot":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] screenshot [--page name] [--path PATH] [--full-page] [--annotate-refs] [--crop x,y,w,h] [--selector CSS] [--aria-role ROLE] [--aria-name NAME] [--nth N] [--padding-px PX] [--timeout-ms MS]\n")
	case "bounds":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] bounds [selector] [--page name] [--aria-role ROLE] [--aria-name NAME] [--nth N] [--timeout-ms MS]\n")
	case "console":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] console [--page name] [--since id] [--limit N] [--level lvl]\n")
	case "save-html":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] save-html [--page name] [--path PATH]\n")
	case "call":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] call <tool> [--args JSON] [--page name]\n")
	case "actions":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] actions [--calls JSON] [--page name] (reads stdin if --calls empty)\n")
	case "wait":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] wait [--page name] [--strategy STR] [--state STATE] [--timeout-ms MS] [--min-wait-ms MS]\n")
	case "list-pages":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] list-pages\n")
	case "close-page":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] close-page <name>\n")
	case "status":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] status\n")
	case "start":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] start\n")
	case "stop":
		fmt.Fprintf(os.Stdout, "Usage: dev-browser-go [globals] stop\n")
	default:
		printUsage()
	}
}

func printDaemonUsage() {
	fmt.Fprintf(os.Stdout, `dev-browser-go --daemon - run daemon only

Usage:
  dev-browser-go --daemon [--profile name] [--host addr] [--port port] [--cdp-port port] [--headless] [--state-file path] [--window-size WxH] [--window-scale SCALE]
`)
}
