package qmd

import (
	"testing"
)

func TestParseSearchResults_WrappedFormat(t *testing.T) {
	input := `{"results":[
		{"docid":"abc123","filepath":"/notes/foo.md","score":0.9,"title":"Foo","snippet":"bar baz"},
		{"docid":"def456","filepath":"/notes/bar.md","score":0.7,"title":"Bar","snippet":""}
	]}`
	results, err := ParseSearchResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].DocID != "abc123" {
		t.Errorf("expected docid 'abc123', got %q", results[0].DocID)
	}
	if results[0].FilePath != "/notes/foo.md" {
		t.Errorf("expected filepath '/notes/foo.md', got %q", results[0].FilePath)
	}
	if results[0].Score != 0.9 {
		t.Errorf("expected score 0.9, got %f", results[0].Score)
	}
}

func TestParseSearchResults_BareArray(t *testing.T) {
	input := `[
		{"docid":"xyz789","filepath":"/docs/hello.md","score":0.5,"title":"Hello","snippet":"world"}
	]`
	results, err := ParseSearchResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].DocID != "xyz789" {
		t.Errorf("expected docid 'xyz789', got %q", results[0].DocID)
	}
}

func TestParseSearchResults_EmptyResults(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty wrapped", `{"results":[]}`},
		{"empty array", `[]`},
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
	// qmd might return {"results": null}
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
	input := `not json at all`
	_, err := ParseSearchResults([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestParseSearchResults_MissingFields(t *testing.T) {
	// Partial fields should not cause an error — missing fields default to zero values
	input := `{"results":[{"docid":"a1b2c3"}]}`
	results, err := ParseSearchResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].FilePath != "" {
		t.Errorf("expected empty filepath, got %q", results[0].FilePath)
	}
}
