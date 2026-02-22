package qmd

import "testing"

func TestBuildSearchArgs_BasicSearch(t *testing.T) {
	c := &Client{}
	args := c.buildSearchArgs("hello world", ModeSearch)

	// Must contain "search" as first arg and "--json"
	if args[0] != "search" {
		t.Errorf("expected first arg 'search', got %q", args[0])
	}
	if !containsArg(args, "--json") {
		t.Error("expected --json flag")
	}
	if !containsArg(args, "hello world") {
		t.Error("expected query in args")
	}
}

func TestBuildSearchArgs_WithCollection(t *testing.T) {
	c := &Client{Collection: "mynotes"}
	args := c.buildSearchArgs("foo", ModeSearch)

	if !containsConsecutive(args, "--collection", "mynotes") {
		t.Error("expected --collection mynotes in args")
	}
}

func TestBuildSearchArgs_WithResults(t *testing.T) {
	c := &Client{Results: 20}
	args := c.buildSearchArgs("foo", ModeSearch)

	if !containsConsecutive(args, "--results", "20") {
		t.Error("expected --results 20 in args")
	}
}

func TestBuildSearchArgs_ModeVSearch(t *testing.T) {
	c := &Client{}
	args := c.buildSearchArgs("foo", ModeVSearch)

	if !containsConsecutive(args, "--mode", "vsearch") {
		t.Error("expected --mode vsearch in args")
	}
}

func TestBuildSearchArgs_ModeQuery(t *testing.T) {
	c := &Client{}
	args := c.buildSearchArgs("foo", ModeQuery)

	if !containsConsecutive(args, "--mode", "query") {
		t.Error("expected --mode query in args")
	}
}

func TestBuildSearchArgs_ModeSearchNoModeFlag(t *testing.T) {
	// Default search mode should NOT add --mode flag
	c := &Client{}
	args := c.buildSearchArgs("foo", ModeSearch)

	if containsArg(args, "--mode") {
		t.Error("search mode should not add --mode flag")
	}
}

func TestBuildSearchArgs_MinScore(t *testing.T) {
	c := &Client{MinScore: 0.75}
	args := c.buildSearchArgs("foo", ModeSearch)

	if !containsConsecutive(args, "--min-score", "0.75") {
		t.Errorf("expected --min-score 0.75, got %v", args)
	}
}

func TestSearchTimeout(t *testing.T) {
	if d := searchTimeout(ModeQuery); d.Seconds() < 15 {
		t.Errorf("query mode timeout should be at least 15s, got %v", d)
	}
	if d := searchTimeout(ModeSearch); d.Seconds() > 20 {
		t.Errorf("search mode timeout should be ≤20s, got %v", d)
	}
}

// helpers

func containsArg(args []string, needle string) bool {
	for _, a := range args {
		if a == needle {
			return true
		}
	}
	return false
}

func containsConsecutive(args []string, a, b string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == a && args[i+1] == b {
			return true
		}
	}
	return false
}
