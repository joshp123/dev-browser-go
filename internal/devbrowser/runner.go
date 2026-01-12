package devbrowser

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type RunResult map[string]interface{}

type ActionsResult struct {
	Results  []map[string]interface{}
	Snapshot string
}

func RunCall(page playwright.Page, name string, args map[string]interface{}, artifactDir string) (RunResult, error) {
	switch name {
	case "goto":
		url, err := requireString(args, "url")
		if err != nil {
			return nil, err
		}
		waitUntil, err := optionalString(args, "wait_until", "domcontentloaded")
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 45_000)
		if err != nil {
			return nil, err
		}
		_, err = page.Goto(url, playwright.PageGotoOptions{
			WaitUntil: getWaitUntil(waitUntil),
			Timeout:   playwright.Float(float64(timeoutMs)),
		})
		if err != nil {
			return nil, err
		}
		return RunResult{"url": page.URL(), "title": safeTitle(page)}, nil

	case "snapshot":
		engine, err := optionalString(args, "engine", "simple")
		if err != nil {
			return nil, err
		}
		format, err := optionalString(args, "format", "list")
		if err != nil {
			return nil, err
		}
		interactiveOnly, err := optionalBool(args, "interactive_only", true)
		if err != nil {
			return nil, err
		}
		includeHeadings, err := optionalBool(args, "include_headings", true)
		if err != nil {
			return nil, err
		}
		maxItems, err := optionalInt(args, "max_items", 80)
		if err != nil {
			return nil, err
		}
		maxChars, err := optionalInt(args, "max_chars", 8000)
		if err != nil {
			return nil, err
		}

		snap, err := GetSnapshot(page, SnapshotOptions{
			Engine:          engine,
			Format:          format,
			InteractiveOnly: interactiveOnly,
			IncludeHeadings: includeHeadings,
			MaxItems:        maxItems,
			MaxChars:        maxChars,
		})
		if err != nil {
			return nil, err
		}
		return RunResult{
			"url":      page.URL(),
			"title":    safeTitle(page),
			"engine":   engine,
			"format":   format,
			"snapshot": snap.Yaml,
			"items":    snap.Items,
		}, nil

	case "click_ref":
		ref, err := requireString(args, "ref")
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 15_000)
		if err != nil {
			return nil, err
		}
		el, err := SelectRef(page, ref, "simple")
		if err != nil {
			return nil, err
		}
		err = el.Click(playwright.ElementHandleClickOptions{Timeout: playwright.Float(float64(timeoutMs))})
		_ = el.Dispose()
		if err != nil {
			return nil, err
		}
		return RunResult{"ref": ref, "clicked": true}, nil

	case "fill_ref":
		ref, err := requireString(args, "ref")
		if err != nil {
			return nil, err
		}
		text, err := requireString(args, "text")
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 15_000)
		if err != nil {
			return nil, err
		}
		el, err := SelectRef(page, ref, "simple")
		if err != nil {
			return nil, err
		}
		err = el.Fill(text, playwright.ElementHandleFillOptions{Timeout: playwright.Float(float64(timeoutMs))})
		_ = el.Dispose()
		if err != nil {
			return nil, err
		}
		return RunResult{"ref": ref, "filled": true}, nil

	case "press":
		key, err := requireString(args, "key")
		if err != nil {
			return nil, err
		}
		if err := page.Keyboard().Press(key); err != nil {
			return nil, err
		}
		return RunResult{"key": key, "pressed": true}, nil

	case "wait":
		strategy, err := optionalString(args, "strategy", "playwright")
		if err != nil {
			return nil, err
		}
		state, err := optionalString(args, "state", "load")
		if err != nil {
			return nil, err
		}
		allowedStates := map[string]bool{"load": true, "domcontentloaded": true, "networkidle": true, "commit": true}
		if !allowedStates[strings.ToLower(state)] {
			return nil, fmt.Errorf("invalid state '%s' (expected one of: load, domcontentloaded, networkidle, commit)", state)
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 10_000)
		if err != nil {
			return nil, err
		}
		minWaitMs, err := optionalInt(args, "min_wait_ms", 0)
		if err != nil {
			return nil, err
		}

		switch strategy {
		case "playwright":
			return waitPlaywright(page, state, timeoutMs, minWaitMs)
		case "perf":
			return waitPerf(page, state, timeoutMs, minWaitMs)
		default:
			return nil, fmt.Errorf("invalid strategy (expected 'playwright' or 'perf')")
		}

	case "screenshot":
		pathArg, err := optionalString(args, "path", "")
		if err != nil {
			return nil, err
		}
		fullPage, err := optionalBool(args, "full_page", true)
		if err != nil {
			return nil, err
		}
		annotate, err := optionalBool(args, "annotate_refs", false)
		if err != nil {
			return nil, err
		}
		crop, err := optionalCrop(args)
		if err != nil {
			return nil, err
		}

		selector, err := optionalString(args, "selector", "")
		if err != nil {
			return nil, err
		}
		ariaRole, err := optionalString(args, "aria_role", "")
		if err != nil {
			return nil, err
		}
		ariaName, err := optionalString(args, "aria_name", "")
		if err != nil {
			return nil, err
		}
		nth, err := optionalInt(args, "nth", 1)
		if err != nil {
			return nil, err
		}
		padding, err := optionalInt(args, "padding_px", 10)
		if err != nil {
			return nil, err
		}
		targetTimeout, err := optionalInt(args, "timeout_ms", 5_000)
		if err != nil {
			return nil, err
		}

		hasTarget := strings.TrimSpace(selector) != "" || strings.TrimSpace(ariaRole) != "" || strings.TrimSpace(ariaName) != ""
		if crop != nil && hasTarget {
			return nil, errors.New("--crop cannot be combined with selector/aria targeting")
		}

		path, err := SafeArtifactPath(artifactDir, pathArg, fmt.Sprintf("screenshot-%d.png", NowMS()))
		if err != nil {
			return nil, err
		}

		opts := playwright.PageScreenshotOptions{Path: playwright.String(path), FullPage: playwright.Bool(fullPage)}
		var clip *playwright.Rect
		var spec TargetSpec

		if hasTarget {
			spec = TargetSpec{Selector: selector, AriaRole: ariaRole, AriaName: ariaName, Nth: nth, Timeout: targetTimeout}
			box, err := resolveBounds(page, spec)
			if err != nil {
				return nil, err
			}
			vp := viewportSize(page)
			clip, err = clipWithPadding(box, padding, vp)
			if err != nil {
				return nil, err
			}
			opts.Clip = clip
			opts.FullPage = playwright.Bool(false)
		}

		if crop != nil {
			opts.Clip = crop
			opts.FullPage = playwright.Bool(false)
		}

		if annotate {
			_ = DrawRefOverlay(page, 80, "simple")
			page.WaitForTimeout(50)
		}
		_, shotErr := page.Screenshot(opts)
		if annotate {
			_ = ClearRefOverlay(page, "simple")
		}
		if shotErr != nil {
			return nil, shotErr
		}

		res := RunResult{"path": path}
		if clip != nil {
			res["selector"] = selector
			res["aria_role"] = ariaRole
			res["aria_name"] = ariaName
			res["nth"] = spec.effectiveNth()
			res["clip"] = map[string]float64{"x": clip.X, "y": clip.Y, "width": clip.Width, "height": clip.Height}
		}
		return res, nil

	case "save_html":
		pathArg, err := optionalString(args, "path", "")
		if err != nil {
			return nil, err
		}
		path, err := SafeArtifactPath(artifactDir, pathArg, fmt.Sprintf("page-%d.html", NowMS()))
		if err != nil {
			return nil, err
		}
		html, err := page.Content()
		if err != nil {
			return nil, err
		}
		if err := osWriteFile(path, []byte(html)); err != nil {
			return nil, err
		}
		return RunResult{"path": path}, nil

	case "bounds":
		selector, err := optionalString(args, "selector", "")
		if err != nil {
			return nil, err
		}
		ariaRole, err := optionalString(args, "aria_role", "")
		if err != nil {
			return nil, err
		}
		ariaName, err := optionalString(args, "aria_name", "")
		if err != nil {
			return nil, err
		}
		nth, err := optionalInt(args, "nth", 1)
		if err != nil {
			return nil, err
		}
		timeoutMs, err := optionalInt(args, "timeout_ms", 5_000)
		if err != nil {
			return nil, err
		}

		spec := TargetSpec{Selector: selector, AriaRole: ariaRole, AriaName: ariaName, Nth: nth, Timeout: timeoutMs}
		box, err := resolveBounds(page, spec)
		if err != nil {
			return nil, err
		}
		return RunResult{
			"selector":  selector,
			"aria_role": ariaRole,
			"aria_name": ariaName,
			"nth":       spec.effectiveNth(),
			"x":         box.X,
			"y":         box.Y,
			"width":     box.Width,
			"height":    box.Height,
		}, nil
	}

	return nil, fmt.Errorf("unknown call '%s'", name)
}

func RunActions(page playwright.Page, calls []map[string]interface{}, artifactDir string) (ActionsResult, error) {
	results := []map[string]interface{}{}
	snapshotText := ""

	for _, call := range calls {
		nameVal, ok := call["name"]
		if !ok {
			return ActionsResult{}, errors.New("each call must include name")
		}
		name, ok := nameVal.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return ActionsResult{}, errors.New("each call must include non-empty string 'name'")
		}
		argsVal, ok := call["arguments"]
		if !ok || argsVal == nil {
			argsVal = map[string]interface{}{}
		}
		args, ok := argsVal.(map[string]interface{})
		if !ok {
			return ActionsResult{}, errors.New("call 'arguments' must be an object")
		}

		res, err := RunCall(page, name, args, artifactDir)
		if err != nil {
			return ActionsResult{}, err
		}
		entry := map[string]interface{}{"name": name, "result": res}
		results = append(results, entry)
		if name == "snapshot" {
			if snap, ok := res["snapshot"].(string); ok {
				snapshotText = snap
			}
		}
	}
	return ActionsResult{Results: results, Snapshot: snapshotText}, nil
}

func getWaitUntil(value string) *playwright.WaitUntilState {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case "commit":
		return playwright.WaitUntilStateCommit
	case "networkidle":
		return playwright.WaitUntilStateNetworkidle
	case "domcontentloaded":
		return playwright.WaitUntilStateDomcontentloaded
	default:
		return playwright.WaitUntilStateLoad
	}
}

func waitPlaywright(page playwright.Page, state string, timeoutMs int, minWaitMs int) (RunResult, error) {
	start := time.Now()
	if minWaitMs > 0 {
		page.WaitForTimeout(float64(minWaitMs))
	}
	var loadState *playwright.LoadState
	switch strings.ToLower(state) {
	case "domcontentloaded", "commit":
		loadState = playwright.LoadStateDomcontentloaded
	case "networkidle":
		loadState = playwright.LoadStateNetworkidle
	default:
		loadState = playwright.LoadStateLoad
	}
	err := page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{State: loadState, Timeout: playwright.Float(float64(timeoutMs))})
	timedOut := isTimeout(err)
	if err != nil && !timedOut {
		return nil, err
	}
	readyState := ""
	if rs, err := page.Evaluate("() => document.readyState"); err == nil {
		if str, ok := rs.(string); ok {
			readyState = str
		}
	}
	waited := int(time.Since(start).Milliseconds())
	if waited < minWaitMs {
		waited = minWaitMs
	}
	return RunResult{
		"ok":          !timedOut,
		"strategy":    "playwright",
		"state":       state,
		"timed_out":   timedOut,
		"waited_ms":   waited,
		"ready_state": readyState,
	}, nil
}

func waitPerf(page playwright.Page, state string, timeoutMs int, minWaitMs int) (RunResult, error) {
	pollInterval := 50 * time.Millisecond
	start := time.Now()
	if minWaitMs > 0 {
		page.WaitForTimeout(float64(minWaitMs))
	}
	deadline := start.Add(time.Duration(timeoutMs) * time.Millisecond)
	lastReady := ""
	lastPending := 0
	success := false

	for time.Now().Before(deadline) {
		data, err := page.Evaluate(perfLoadStateJS)
		if err == nil {
			if m, ok := data.(map[string]interface{}); ok {
				if rs, ok := m["readyState"].(string); ok {
					lastReady = rs
				}
				if pending, ok := asInt(m["pendingRequests"]); ok {
					lastPending = pending
				}
			}
		}

		if lastReady != "" && readyStateSatisfies(lastReady, state) && lastPending == 0 {
			success = true
			break
		}
		page.WaitForTimeout(float64(pollInterval.Milliseconds()))
	}

	waited := int(time.Since(start).Milliseconds())
	return RunResult{
		"ok":               success,
		"strategy":         "perf",
		"state":            state,
		"timed_out":        !success,
		"waited_ms":        waited,
		"ready_state":      lastReady,
		"pending_requests": lastPending,
	}, nil
}

const perfLoadStateJS = `() => {
  const doc = globalThis.document;
  const perf = globalThis.performance;
  const readyState = doc && typeof doc.readyState === "string" ? doc.readyState : "unknown";
  if (!perf || typeof perf.getEntriesByType !== "function" || typeof perf.now !== "function") {
    return { readyState, pendingRequests: 0 };
  }

  const now = perf.now();
  const resources = perf.getEntriesByType("resource") || [];

  const adPatterns = [
    "doubleclick.net",
    "googlesyndication.com",
    "googletagmanager.com",
    "google-analytics.com",
    "facebook.net",
    "connect.facebook.net",
    "analytics",
    "ads",
    "tracking",
    "pixel",
    "hotjar.com",
    "clarity.ms",
    "mixpanel.com",
    "segment.com",
    "newrelic.com",
    "nr-data.net",
    "/tracker/",
    "/collector/",
    "/beacon/",
    "/telemetry/",
    "/log/",
    "/events/",
    "/track.",
    "/metrics/",
  ];

  const nonCriticalTypes = ["img", "image", "icon", "font"];

  let pending = 0;
  for (const entry of resources) {
    if (!entry || entry.responseEnd !== 0) continue;
    const url = String(entry.name || "");

    if (!url || url.startsWith("data:") || url.length > 500) continue;
    if (adPatterns.some((p) => url.includes(p))) continue;

    const loadingDuration = now - (entry.startTime || 0);
    if (loadingDuration > 10000) continue;

    const resourceType = String(entry.initiatorType || "unknown");
    if (nonCriticalTypes.includes(resourceType) && loadingDuration > 3000) continue;

    const isImageUrl = /\.(jpg|jpeg|png|gif|webp|svg|ico)(\?|$)/i.test(url);
    if (isImageUrl && loadingDuration > 3000) continue;

    pending++;
  }
  return { readyState, pendingRequests: pending };
}`

func readyStateSatisfies(ready string, state string) bool {
	rs := strings.ToLower(ready)
	if state == "domcontentloaded" || state == "commit" {
		return rs == "interactive" || rs == "complete"
	}
	return rs == "complete"
}

func requireString(args map[string]interface{}, key string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return "", fmt.Errorf("expected non-empty string '%s'", key)
	}
	str, ok := raw.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", fmt.Errorf("expected non-empty string '%s'", key)
	}
	return str, nil
}

func optionalString(args map[string]interface{}, key string, def string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	str, ok := raw.(string)
	if !ok || strings.TrimSpace(str) == "" {
		return "", fmt.Errorf("expected string '%s'", key)
	}
	return str, nil
}

func optionalBool(args map[string]interface{}, key string, def bool) (bool, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	b, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("expected boolean '%s'", key)
	}
	return b, nil
}

func optionalInt(args map[string]interface{}, key string, def int) (int, error) {
	raw, ok := args[key]
	if !ok {
		return def, nil
	}
	if i, ok := asInt(raw); ok {
		if i < 0 {
			return 0, fmt.Errorf("expected non-negative integer '%s'", key)
		}
		return i, nil
	}
	return 0, fmt.Errorf("expected non-negative integer '%s'", key)
}

func asInt(v interface{}) (int, bool) {
	switch t := v.(type) {
	case int:
		return t, true
	case int64:
		return int(t), true
	case float64:
		return int(t), true
	case float32:
		return int(t), true
	default:
		return 0, false
	}
}

func optionalCrop(args map[string]interface{}) (*playwright.Rect, error) {
	raw, ok := args["crop"]
	if !ok || raw == nil {
		return nil, nil
	}

	toVals := func(seq []interface{}) ([]int, error) {
		if len(seq) != 4 {
			return nil, errors.New("crop must have 4 items: x,y,width,height")
		}
		vals := make([]int, 4)
		for i, v := range seq {
			n, ok := asInt(v)
			if !ok || n < 0 {
				return nil, errors.New("crop values must be non-negative integers")
			}
			vals[i] = n
		}
		if vals[2] < 1 || vals[3] < 1 {
			return nil, errors.New("crop width/height must be positive")
		}
		if vals[2] > 2000 {
			vals[2] = 2000
		}
		if vals[3] > 2000 {
			vals[3] = 2000
		}
		return vals, nil
	}

	var vals []int
	switch t := raw.(type) {
	case string:
		parts := strings.Split(t, ",")
		if len(parts) != 4 {
			return nil, errors.New("--crop must be x,y,width,height")
		}
		seq := make([]interface{}, 0, 4)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				return nil, errors.New("--crop must be x,y,width,height")
			}
			seq = append(seq, p)
		}
		valsSeq := make([]interface{}, 0, 4)
		for _, s := range seq {
			num, err := strconv.Atoi(s.(string))
			if err != nil {
				return nil, errors.New("crop values must be integers")
			}
			valsSeq = append(valsSeq, num)
		}
		var err error
		vals, err = toVals(valsSeq)
		if err != nil {
			return nil, err
		}
	case map[string]interface{}:
		seq := []interface{}{t["x"], t["y"], t["width"], t["height"]}
		v, err := toVals(seq)
		if err != nil {
			return nil, err
		}
		vals = v
	case []interface{}:
		v, err := toVals(t)
		if err != nil {
			return nil, err
		}
		vals = v
	default:
		return nil, errors.New("crop must be string, array, or object")
	}

	return &playwright.Rect{X: float64(vals[0]), Y: float64(vals[1]), Width: float64(vals[2]), Height: float64(vals[3])}, nil
}

func safeTitle(page playwright.Page) string {
	if page == nil {
		return ""
	}
	if title, err := page.Title(); err == nil {
		return title
	}
	return ""
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
