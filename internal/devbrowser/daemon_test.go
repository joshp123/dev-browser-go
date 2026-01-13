package devbrowser

import "testing"

func TestSelectConsoleLogs_SinceAndLimit(t *testing.T) {
	allEntries := []ConsoleEntry{
		{ID: 1, Type: "debug"},
		{ID: 2, Type: "info"},
		{ID: 3, Type: "warning"},
		{ID: 4, Type: "error"},
		{ID: 5, Type: "pageerror"},
		{ID: 6, Type: "log"},
	}
	filter, err := parseConsoleLevels("info,error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	afterSince := []ConsoleEntry{allEntries[3], allEntries[4], allEntries[5]}
	withSince := selectConsoleLogs(afterSince, filter, 3, 2)
	if len(withSince) != 2 {
		t.Fatalf("expected 2 entries with since, got %d", len(withSince))
	}
	if withSince[0].ID != 4 || withSince[1].ID != 5 {
		t.Fatalf("unexpected entries with since: %+v", withSince)
	}

	noSince := selectConsoleLogs(allEntries, filter, 0, 2)
	if len(noSince) != 2 {
		t.Fatalf("expected 2 entries without since, got %d", len(noSince))
	}
	if noSince[0].ID != 5 || noSince[1].ID != 6 {
		t.Fatalf("unexpected entries without since: %+v", noSince)
	}
}
