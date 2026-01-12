package devbrowser

import (
	"testing"
)

// Tests for parseConsoleLevels (Fix #7)

func TestParseConsoleLevels_All(t *testing.T) {
	filter, err := parseConsoleLevels("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filter.all {
		t.Fatalf("expected all=true")
	}
}

func TestParseConsoleLevels_Empty(t *testing.T) {
	filter, err := parseConsoleLevels("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty should use default: info,warning,error
	if !filter.allowed["info"] || !filter.allowed["warning"] || !filter.allowed["error"] {
		t.Fatalf("expected default levels (info,warning,error), got %+v", filter.allowed)
	}
	if filter.allowed["debug"] {
		t.Fatalf("debug should not be in default levels")
	}
}

func TestParseConsoleLevels_Whitespace(t *testing.T) {
	filter, err := parseConsoleLevels("  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Whitespace should use default
	if !filter.allowed["info"] || !filter.allowed["warning"] || !filter.allowed["error"] {
		t.Fatalf("expected default levels, got %+v", filter.allowed)
	}
}

func TestParseConsoleLevels_InvalidInput(t *testing.T) {
	tests := []string{
		"invalid",
		"foo,bar",
		"info,invalid",
		",,",
	}
	for _, input := range tests {
		_, err := parseConsoleLevels(input)
		if err == nil {
			t.Fatalf("expected error for input %q", input)
		}
	}
}

func TestParseConsoleLevels_AliasWarn(t *testing.T) {
	filter, err := parseConsoleLevels("warn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filter.allowed["warning"] {
		t.Fatalf("expected 'warn' alias to map to 'warning'")
	}
	if len(filter.allowed) != 1 {
		t.Fatalf("expected only 'warning', got %+v", filter.allowed)
	}
}

func TestParseConsoleLevels_AliasWarnings(t *testing.T) {
	filter, err := parseConsoleLevels("warnings")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filter.allowed["warning"] {
		t.Fatalf("expected 'warnings' alias to map to 'warning'")
	}
}

func TestParseConsoleLevels_AliasErrors(t *testing.T) {
	filter, err := parseConsoleLevels("errors")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filter.allowed["error"] {
		t.Fatalf("expected 'errors' alias to map to 'error'")
	}
}

func TestParseConsoleLevels_ValidInputs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"debug", []string{"debug"}},
		{"info", []string{"info"}},
		{"warning", []string{"warning"}},
		{"error", []string{"error"}},
		{"debug,info", []string{"debug", "info"}},
		{"info,warning,error", []string{"info", "warning", "error"}},
		{"debug,info,warning,error", []string{"debug", "info", "warning", "error"}},
	}
	for _, tt := range tests {
		filter, err := parseConsoleLevels(tt.input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.input, err)
		}
		if filter.all {
			t.Fatalf("expected all=false for %q", tt.input)
		}
		for _, level := range tt.expected {
			if !filter.allowed[level] {
				t.Fatalf("expected %q to be allowed in filter for input %q", level, tt.input)
			}
		}
		if len(filter.allowed) != len(tt.expected) {
			t.Fatalf("expected %d levels for %q, got %d: %+v", len(tt.expected), tt.input, len(filter.allowed), filter.allowed)
		}
	}
}

func TestParseConsoleLevels_MixedValidInvalid(t *testing.T) {
	_, err := parseConsoleLevels("info,invalid,error")
	if err == nil {
		t.Fatalf("expected error for mixed valid/invalid input")
	}
}

func TestParseConsoleLevels_CaseSensitivity(t *testing.T) {
	tests := []string{
		"INFO",
		"Warning",
		"ERROR",
		"DeBuG",
		"Info,Warning,Error",
	}
	for _, input := range tests {
		filter, err := parseConsoleLevels(input)
		if err != nil {
			t.Fatalf("unexpected error for case-insensitive input %q: %v", input, err)
		}
		if len(filter.allowed) == 0 && !filter.all {
			t.Fatalf("expected valid filter for case-insensitive input %q", input)
		}
	}
}

func TestParseConsoleLevels_CommaSeparated(t *testing.T) {
	filter, err := parseConsoleLevels("debug,info,warning,error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"debug", "info", "warning", "error"}
	for _, level := range expected {
		if !filter.allowed[level] {
			t.Fatalf("expected %q to be allowed", level)
		}
	}
	if len(filter.allowed) != len(expected) {
		t.Fatalf("expected %d levels, got %d", len(expected), len(filter.allowed))
	}
}

func TestParseConsoleLevels_WithSpaces(t *testing.T) {
	filter, err := parseConsoleLevels(" info , warning , error ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filter.allowed["info"] || !filter.allowed["warning"] || !filter.allowed["error"] {
		t.Fatalf("expected info, warning, error to be allowed, got %+v", filter.allowed)
	}
}

func TestFilterConsoleEntries_All(t *testing.T) {
	entries := []ConsoleEntry{
		{Type: "debug"},
		{Type: "info"},
		{Type: "warning"},
		{Type: "error"},
	}
	filter, err := parseConsoleLevels("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := filterConsoleEntries(entries, filter)
	if len(out) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(out))
	}
}

func TestFilterConsoleEntries_SpecificLevels(t *testing.T) {
	entries := []ConsoleEntry{
		{Type: "debug"},
		{Type: "log"},
		{Type: "warning"},
		{Type: "pageerror"},
		{Type: "error"},
	}
	filter, err := parseConsoleLevels("info,error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := filterConsoleEntries(entries, filter)
	if len(out) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(out))
	}
	if out[0].Type != "log" || out[1].Type != "pageerror" || out[2].Type != "error" {
		t.Fatalf("unexpected filter order: %+v", out)
	}
}

func TestFilterConsoleEntries_EmptyFilter(t *testing.T) {
	entries := []ConsoleEntry{{Type: "info"}}
	out := filterConsoleEntries(entries, consoleLevelFilter{allowed: map[string]bool{}})
	if len(out) != 0 {
		t.Fatalf("expected 0 entries with empty filter, got %d", len(out))
	}
}

func TestConsoleLevelForType(t *testing.T) {
	tests := []struct {
		msgType string
		want    string
	}{
		{"log", "info"},
		{"pageerror", "error"},
		{"warn", "warning"},
		{"warning", "warning"},
		{"ERROR", "error"},
		{"unknown", "info"},
	}
	for _, tt := range tests {
		t.Run(tt.msgType, func(t *testing.T) {
			if got := consoleLevelForType(tt.msgType); got != tt.want {
				t.Fatalf("expected %q for %q, got %q", tt.want, tt.msgType, got)
			}
		})
	}
}

// Tests for consoleStore.list (Fix #8)

func TestConsoleStore_List_SinceZero(t *testing.T) {
	store := newConsoleStore(10)
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg1"})
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg2"})
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg3"})

	logs, lastID := store.list("page1", 0, 0)
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs with since=0, got %d", len(logs))
	}
	if logs[0].Text != "msg1" || logs[1].Text != "msg2" || logs[2].Text != "msg3" {
		t.Fatalf("unexpected log order or content")
	}
	if lastID != logs[2].ID {
		t.Fatalf("expected lastID to match last entry ID")
	}
}

func TestConsoleStore_List_SinceNonExistent(t *testing.T) {
	store := newConsoleStore(10)
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg1"})
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg2"})

	logs, lastID := store.list("page1", 9999, 0)
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs for non-existent since ID, got %d", len(logs))
	}
	if lastID != 9999 {
		t.Fatalf("expected lastID to be since value when no logs returned, got %d", lastID)
	}
}

func TestConsoleStore_List_LimitWithoutSince(t *testing.T) {
	store := newConsoleStore(10)
	for i := 0; i < 5; i++ {
		store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg"})
	}

	logs, _ := store.list("page1", 0, 3)
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs with limit=3, got %d", len(logs))
	}
	// Should get the last 3 entries
	allLogs, _ := store.list("page1", 0, 0)
	for i := 0; i < 3; i++ {
		if logs[i].ID != allLogs[i+2].ID {
			t.Fatalf("expected last 3 entries, got different IDs")
		}
	}
}

func TestConsoleStore_List_LimitWithSince(t *testing.T) {
	store := newConsoleStore(10)
	var firstID int64
	for i := 0; i < 5; i++ {
		store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg"})
		if i == 0 {
			logs, _ := store.list("page1", 0, 0)
			firstID = logs[0].ID
		}
	}

	logs, _ := store.list("page1", firstID, 2)
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs with since=%d and limit=2, got %d", firstID, len(logs))
	}
	// Should get entries after firstID, limited to 2
	if logs[0].ID <= firstID {
		t.Fatalf("expected first log ID to be > since")
	}
}

func TestConsoleStore_List_EmptyBuffer(t *testing.T) {
	store := newConsoleStore(10)
	logs, lastID := store.list("page1", 0, 0)
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs for empty buffer, got %d", len(logs))
	}
	if lastID != 0 {
		t.Fatalf("expected lastID=0 for empty buffer, got %d", lastID)
	}
}

func TestConsoleStore_List_EmptyBufferWithSince(t *testing.T) {
	store := newConsoleStore(10)
	logs, lastID := store.list("page1", 5, 0)
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs for empty buffer, got %d", len(logs))
	}
	if lastID != 5 {
		t.Fatalf("expected lastID=5 (since value) for empty buffer, got %d", lastID)
	}
}

func TestConsoleStore_List_StartExceedsLength(t *testing.T) {
	store := newConsoleStore(10)
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg1"})
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg2"})

	allLogs, _ := store.list("page1", 0, 0)
	lastID := allLogs[len(allLogs)-1].ID

	// Request logs after the last ID
	logs, returnedLastID := store.list("page1", lastID, 0)
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs when since points to last ID, got %d", len(logs))
	}
	if returnedLastID != lastID {
		t.Fatalf("expected lastID to remain %d, got %d", lastID, returnedLastID)
	}
}

func TestConsoleStore_List_VariousSinceLimitCombinations(t *testing.T) {
	store := newConsoleStore(10)
	for i := 0; i < 10; i++ {
		store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg"})
	}

	allLogs, _ := store.list("page1", 0, 0)
	if len(allLogs) != 10 {
		t.Fatalf("expected 10 logs, got %d", len(allLogs))
	}

	tests := []struct {
		name          string
		since         int64
		limit         int
		expectedCount int
	}{
		{"no since, no limit", 0, 0, 10},
		{"no since, limit 5", 0, 5, 5},
		{"since middle, no limit", allLogs[4].ID, 0, 5},
		{"since middle, limit 2", allLogs[4].ID, 2, 2},
		{"since first, limit 3", allLogs[0].ID, 3, 3},
		{"since last-1, no limit", allLogs[8].ID, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, _ := store.list("page1", tt.since, tt.limit)
			if len(logs) != tt.expectedCount {
				t.Fatalf("expected %d logs, got %d", tt.expectedCount, len(logs))
			}
		})
	}
}

func TestConsoleStore_List_MultiplePages(t *testing.T) {
	store := newConsoleStore(10)
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg1"})
	store.appendEntry("page2", ConsoleEntry{Type: "info", Text: "msg2"})
	store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg3"})

	logs1, _ := store.list("page1", 0, 0)
	logs2, _ := store.list("page2", 0, 0)

	if len(logs1) != 2 {
		t.Fatalf("expected 2 logs for page1, got %d", len(logs1))
	}
	if len(logs2) != 1 {
		t.Fatalf("expected 1 log for page2, got %d", len(logs2))
	}
	if logs1[0].Text != "msg1" || logs1[1].Text != "msg3" {
		t.Fatalf("unexpected logs for page1")
	}
	if logs2[0].Text != "msg2" {
		t.Fatalf("unexpected log for page2")
	}
}

func TestConsoleStore_List_BufferWrapping(t *testing.T) {
	store := newConsoleStore(5)
	// Add more entries than max to test buffer wrapping
	for i := 0; i < 10; i++ {
		store.appendEntry("page1", ConsoleEntry{Type: "info", Text: "msg"})
	}

	logs, _ := store.list("page1", 0, 0)
	if len(logs) != 5 {
		t.Fatalf("expected 5 logs after buffer wrapping, got %d", len(logs))
	}
	// IDs should be sequential from 6 to 10
	if logs[0].ID != 6 || logs[4].ID != 10 {
		t.Fatalf("unexpected IDs after buffer wrapping: first=%d, last=%d", logs[0].ID, logs[4].ID)
	}
}
