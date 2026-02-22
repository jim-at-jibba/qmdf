package tui

import (
	"testing"

	"github.com/jim-at-jibba/qmdf/internal/qmd"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		input    string
		n        int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell…"},
		{"hello", 5, "hello"},
		{"hi", 1, "…"},
		{"", 10, ""},
	}
	for _, tc := range cases {
		got := truncate(tc.input, tc.n)
		if got != tc.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.input, tc.n, got, tc.expected)
		}
	}
}

func TestCycleMode(t *testing.T) {
	if cycleMode(qmd.ModeSearch) != qmd.ModeVSearch {
		t.Error("search → vsearch")
	}
	if cycleMode(qmd.ModeVSearch) != qmd.ModeQuery {
		t.Error("vsearch → query")
	}
	if cycleMode(qmd.ModeQuery) != qmd.ModeSearch {
		t.Error("query → search")
	}
}

func TestModeLabel(t *testing.T) {
	cases := map[qmd.Mode]string{
		qmd.ModeSearch:  "SEARCH",
		qmd.ModeVSearch: "VSEARCH",
		qmd.ModeQuery:   "QUERY",
	}
	for mode, expected := range cases {
		if got := modeLabel(mode); got != expected {
			t.Errorf("modeLabel(%q) = %q, want %q", mode, got, expected)
		}
	}
}

func TestFormatElapsed(t *testing.T) {
	if got := formatElapsed(500); got != "500ms" {
		t.Errorf("expected '500ms', got %q", got)
	}
	if got := formatElapsed(1500); got != "1.5s" {
		t.Errorf("expected '1.5s', got %q", got)
	}
}

func TestRenderEmptyState(t *testing.T) {
	// Smoke test — just ensure no panic
	_ = renderEmptyState(false, "", 80, 20)
	_ = renderEmptyState(true, "query", 80, 20)
	_ = renderEmptyState(false, "query", 80, 20)
}

func TestRenderResultList_NoResults(t *testing.T) {
	out := renderResultList(nil, 0, 40, 20, false, "")
	if out == "" {
		t.Error("expected non-empty output for empty state")
	}
}

func TestRenderResultList_WithResults(t *testing.T) {
	results := []qmd.SearchResult{
		{DocID: "#a1b2c3", File: "qmd://col/notes/one.md", Score: 0.9, Title: "One"},
		{DocID: "#d4e5f6", File: "qmd://col/notes/two.md", Score: 0.7, Title: "Two"},
	}
	out := renderResultList(results, 0, 60, 20, false, "one")
	if out == "" {
		t.Error("expected non-empty output with results")
	}
}

func TestPreviewCache(t *testing.T) {
	c := newPreviewCache()

	// Miss
	if _, ok := c.get("missing"); ok {
		t.Error("expected cache miss")
	}

	// Set and hit
	c.set("doc1", "content1")
	if v, ok := c.get("doc1"); !ok || v != "content1" {
		t.Error("expected cache hit for doc1")
	}

	// Overflow: fill past max and ensure oldest is evicted
	for i := 0; i < previewCacheMax+5; i++ {
		c.set(string(rune('a'+i)), "x")
	}
	if len(c.data) > previewCacheMax {
		t.Errorf("cache grew beyond max: %d entries", len(c.data))
	}
}
