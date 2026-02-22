package qmd

import "testing"

func TestBuildSearchArgs_BasicSearch(t *testing.T) {
	c := &Client{}
	args := c.buildSearchArgs("hello world", ModeSearch)

	// First arg must be the subcommand "search", not a flag
	if args[0] != "search" {
		t.Errorf("expected subcommand 'search', got %q", args[0])
	}
	if !containsArg(args, "--json") {
		t.Error("expected --json flag")
	}
	if !containsArg(args, "hello world") {
		t.Error("expected query in args")
	}
	// Must NOT use --mode flag
	if containsArg(args, "--mode") {
		t.Error("should not use --mode flag; mode is a subcommand")
	}
}

func TestBuildSearchArgs_VSearch(t *testing.T) {
	c := &Client{}
	args := c.buildSearchArgs("foo", ModeVSearch)

	if args[0] != "vsearch" {
		t.Errorf("expected subcommand 'vsearch', got %q", args[0])
	}
	if containsArg(args, "--mode") {
		t.Error("should not use --mode flag")
	}
}

func TestBuildSearchArgs_Query(t *testing.T) {
	c := &Client{}
	args := c.buildSearchArgs("foo", ModeQuery)

	if args[0] != "query" {
		t.Errorf("expected subcommand 'query', got %q", args[0])
	}
}

func TestBuildSearchArgs_WithCollection(t *testing.T) {
	c := &Client{Collection: "mynotes"}
	args := c.buildSearchArgs("foo", ModeSearch)

	if !containsConsecutive(args, "-c", "mynotes") {
		t.Errorf("expected -c mynotes in args, got %v", args)
	}
}

func TestBuildSearchArgs_WithResults(t *testing.T) {
	c := &Client{Results: 20}
	args := c.buildSearchArgs("foo", ModeSearch)

	// Results flag is -n, not --results
	if !containsConsecutive(args, "-n", "20") {
		t.Errorf("expected -n 20 in args, got %v", args)
	}
	if containsArg(args, "--results") {
		t.Error("should use -n not --results")
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
	if d := searchTimeout(ModeQuery); d.Seconds() < 30 {
		t.Errorf("query mode timeout should be >= 30s, got %v", d)
	}
	if d := searchTimeout(ModeVSearch); d.Seconds() < 15 {
		t.Errorf("vsearch mode timeout should be >= 15s, got %v", d)
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
