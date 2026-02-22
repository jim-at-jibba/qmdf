package qmd

import (
	"testing"
)

func TestParseSearchResults_BareArray(t *testing.T) {
	// This is qmd's actual output format
	input := `[
		{"docid":"#5ada41","file":"qmd://sidekick/knowledge-base/feature-flags.md","score":0.63,"title":"Feature Flags","context":"Work docs","snippet":"snippet text"},
		{"docid":"#fbf924","file":"qmd://sidekick/apps/running-locally.md","score":0.51,"title":"Running locally","context":"Work docs","snippet":""}
	]`
	results, err := ParseSearchResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].DocID != "#5ada41" {
		t.Errorf("expected docid '#5ada41', got %q", results[0].DocID)
	}
	if results[0].File != "qmd://sidekick/knowledge-base/feature-flags.md" {
		t.Errorf("unexpected file: %q", results[0].File)
	}
	if results[0].Score != 0.63 {
		t.Errorf("expected score 0.63, got %f", results[0].Score)
	}
}

func TestParseSearchResults_WrappedFormat(t *testing.T) {
	// Accept wrapped format too for defensiveness
	input := `{"results":[
		{"docid":"#abc123","file":"qmd://col/foo.md","score":0.9,"title":"Foo","snippet":"bar baz"}
	]}`
	results, err := ParseSearchResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].DocID != "#abc123" {
		t.Errorf("expected '#abc123', got %q", results[0].DocID)
	}
}

func TestParseSearchResults_EmptyResults(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty array", `[]`},
		{"empty wrapped", `{"results":[]}`},
		{"empty bytes", ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := ParseSearchResults([]byte(tc.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(results) != 0 {
				t.Errorf("expected 0 results, got %d", len(results))
			}
		})
	}
}

func TestParseSearchResults_NullResults(t *testing.T) {
	input := `{"results":null}`
	results, err := ParseSearchResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseSearchResults_InvalidJSON(t *testing.T) {
	_, err := ParseSearchResults([]byte(`not json at all`))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDisplayPath(t *testing.T) {
	cases := []struct {
		file     string
		expected string
	}{
		{"qmd://sidekick/knowledge-base/file.md", "knowledge-base/file.md"},
		{"qmd://sidekick/file.md", "file.md"},
		{"qmd://col/a/b/c.md", "a/b/c.md"},
		{"/real/path/file.md", "/real/path/file.md"}, // non-qmd URI passed through
	}
	for _, tc := range cases {
		r := SearchResult{File: tc.file}
		if got := r.DisplayPath(); got != tc.expected {
			t.Errorf("DisplayPath(%q) = %q, want %q", tc.file, got, tc.expected)
		}
	}
}
