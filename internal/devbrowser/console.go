package devbrowser

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/playwright-community/playwright-go"
)

const defaultConsoleLogMax = 200
const defaultConsoleLevels = "info,warning,error"

type ConsoleEntry struct {
	ID     int64  `json:"id"`
	TimeMS int64  `json:"time_ms"`
	Type   string `json:"type"`
	Text   string `json:"text"`
	URL    string `json:"url,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

type consoleStore struct {
	mu     sync.Mutex
	logs   map[string][]ConsoleEntry
	max    int
	nextID int64
}

type consoleLevelFilter struct {
	allowed map[string]bool
	all     bool
}

func newConsoleStore(max int) *consoleStore {
	if max <= 0 {
		max = defaultConsoleLogMax
	}
	return &consoleStore{
		logs: make(map[string][]ConsoleEntry),
		max:  max,
	}
}

func (c *consoleStore) append(name string, msg playwright.ConsoleMessage) {
	entry := ConsoleEntry{
		Type: msg.Type(),
		Text: msg.Text(),
	}
	if loc := msg.Location(); loc != nil {
		entry.URL = loc.URL
		entry.Line = loc.LineNumber + 1
		entry.Column = loc.ColumnNumber + 1
	}
	c.appendEntry(name, entry)
}

func (c *consoleStore) appendPageError(name string, err error) {
	if err == nil {
		return
	}
	c.appendEntry(name, ConsoleEntry{
		Type: "pageerror",
		Text: err.Error(),
	})
}

func (c *consoleStore) appendEntry(name string, entry ConsoleEntry) {
	entry.ID = atomic.AddInt64(&c.nextID, 1)
	if entry.TimeMS == 0 {
		entry.TimeMS = NowMS()
	}

	c.mu.Lock()
	logs := c.logs[name]
	if c.max > 0 && len(logs) >= c.max {
		logs = logs[len(logs)-c.max+1:]
	}
	logs = append(logs, entry)
	c.logs[name] = logs
	c.mu.Unlock()
}

func (c *consoleStore) list(name string, since int64, limit int) ([]ConsoleEntry, int64) {
	c.mu.Lock()
	logs := c.logs[name]
	entries := logs
	if since > 0 {
		start := len(logs)
		for i, entry := range logs {
			if entry.ID > since {
				start = i
				break
			}
		}
		if start >= len(logs) {
			entries = nil
		} else {
			entries = logs[start:]
		}
		if limit > 0 && len(entries) > limit {
			entries = entries[:limit]
		}
	} else if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	out := make([]ConsoleEntry, len(entries))
	copy(out, entries)
	lastID := int64(0)
	if len(out) > 0 {
		lastID = out[len(out)-1].ID
	} else if since > 0 {
		lastID = since
	}
	c.mu.Unlock()
	return out, lastID
}

func (c *consoleStore) clear(name string) {
	c.mu.Lock()
	delete(c.logs, name)
	c.mu.Unlock()
}

func (c *consoleStore) clearAll() {
	c.mu.Lock()
	c.logs = make(map[string][]ConsoleEntry)
	atomic.StoreInt64(&c.nextID, 0)
	c.mu.Unlock()
}

func parseConsoleLevels(raw string) (consoleLevelFilter, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = defaultConsoleLevels
	}
	filter := consoleLevelFilter{allowed: make(map[string]bool)}
	for _, part := range strings.Split(raw, ",") {
		level := strings.ToLower(strings.TrimSpace(part))
		if level == "" {
			continue
		}
		switch level {
		case "all":
			filter.all = true
			return filter, nil
		case "warn", "warnings":
			level = "warning"
		case "errors":
			level = "error"
		}
		switch level {
		case "debug", "info", "warning", "error":
			filter.allowed[level] = true
		default:
			return consoleLevelFilter{}, errors.New("invalid levels (expected debug, info, warning, error, all)")
		}
	}
	if len(filter.allowed) == 0 && !filter.all {
		return consoleLevelFilter{}, errors.New("invalid levels (expected debug, info, warning, error, all)")
	}
	return filter, nil
}

func filterConsoleEntries(entries []ConsoleEntry, filter consoleLevelFilter) []ConsoleEntry {
	if filter.all {
		return entries
	}
	out := make([]ConsoleEntry, 0, len(entries))
	for _, entry := range entries {
		level := consoleLevelForType(entry.Type)
		if filter.allowed[level] {
			out = append(out, entry)
		}
	}
	return out
}

func consoleLevelForType(msgType string) string {
	switch strings.ToLower(strings.TrimSpace(msgType)) {
	case "debug":
		return "debug"
	case "error":
		return "error"
	case "warning", "warn":
		return "warning"
	case "pageerror":
		return "error"
	case "info", "log":
		return "info"
	case "dir", "dirxml", "table", "trace", "clear", "startgroup", "startgroupcollapsed", "endgroup", "assert", "profile", "profileend", "count", "timeend":
		return "info"
	default:
		return "info"
	}
}
