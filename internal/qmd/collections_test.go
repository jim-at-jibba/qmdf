package qmd

import "testing"

func TestParseCollections_MultiLineFields(t *testing.T) {
	// Typical qmd output: name on its own line, fields indented below
	input := `Configured Collections
sidekick
  Pattern: **/*.md
  Files: 34
  Updated: 4d ago`

	results := parseCollections(input)
	if len(results) != 1 {
		t.Fatalf("expected 1 collection, got %d: %+v", len(results), results)
	}
	c := results[0]
	if c.Name != "sidekick" {
		t.Errorf("name: got %q, want 'sidekick'", c.Name)
	}
	if c.Pattern != "**/*.md" {
		t.Errorf("pattern: got %q, want '**/*.md'", c.Pattern)
	}
	if c.FileCount != 34 {
		t.Errorf("filecount: got %d, want 34", c.FileCount)
	}
	if c.Updated != "4d ago" {
		t.Errorf("updated: got %q, want '4d ago'", c.Updated)
	}
}

func TestParseCollections_EmptyFields(t *testing.T) {
	// Empty field labels (Pattern: with no value) must not become collection names
	input := `sidekick
Pattern:
Files:
Updated:`

	results := parseCollections(input)
	if len(results) != 1 {
		t.Fatalf("expected 1 collection, got %d: %+v", len(results), results)
	}
	if results[0].Name != "sidekick" {
		t.Errorf("name: got %q, want 'sidekick'", results[0].Name)
	}
}

func TestParseCollections_InlineParens(t *testing.T) {
	// Inline parens format
	input := `sidekick (34 files, **/*.md, updated 4d ago)`

	results := parseCollections(input)
	if len(results) != 1 {
		t.Fatalf("expected 1 collection, got %d: %+v", len(results), results)
	}
	c := results[0]
	if c.Name != "sidekick" {
		t.Errorf("name: got %q, want 'sidekick'", c.Name)
	}
	if c.FileCount != 34 {
		t.Errorf("filecount: got %d, want 34", c.FileCount)
	}
}

func TestParseCollections_Multiple(t *testing.T) {
	input := `Configured Collections
sidekick
  Files: 34
my-notes
  Files: 120`

	results := parseCollections(input)
	if len(results) != 2 {
		t.Fatalf("expected 2 collections, got %d: %+v", len(results), results)
	}
	if results[0].Name != "sidekick" || results[0].FileCount != 34 {
		t.Errorf("first collection wrong: %+v", results[0])
	}
	if results[1].Name != "my-notes" || results[1].FileCount != 120 {
		t.Errorf("second collection wrong: %+v", results[1])
	}
}

func TestParseContexts_InlineFormat(t *testing.T) {
	// Inline "path: text" on one line
	input := `Configured Contexts
sidekick
  / (root): Work documentation and notes
  /projects: Code and project files`

	results := parseContexts(input)
	if len(results) != 2 {
		t.Fatalf("expected 2 contexts, got %d: %+v", len(results), results)
	}
	if results[0].Path != "/ (root)" || results[0].Text != "Work documentation and notes" {
		t.Errorf("first context wrong: %+v", results[0])
	}
	if results[1].Path != "/projects" || results[1].Text != "Code and project files" {
		t.Errorf("second context wrong: %+v", results[1])
	}
}

func TestParseContexts_TwoLineFormat(t *testing.T) {
	// Actual qmd format: path on one line, text on the next
	input := `Configured Contexts
sidekick
/ (root)
Work documentation and notes
/ (root)
Learning notes and TIL`

	results := parseContexts(input)
	if len(results) != 2 {
		t.Fatalf("expected 2 contexts, got %d: %+v", len(results), results)
	}
	if results[0].Path != "/ (root)" || results[0].Text != "Work documentation and notes" {
		t.Errorf("first context wrong: %+v", results[0])
	}
	if results[1].Path != "/ (root)" || results[1].Text != "Learning notes and TIL" {
		t.Errorf("second context wrong: %+v", results[1])
	}
}

func TestParseContexts_Empty(t *testing.T) {
	results := parseContexts("")
	if len(results) != 0 {
		t.Errorf("expected 0 contexts for empty input, got %d", len(results))
	}
}

func TestParseContexts_SingleWordDescription(t *testing.T) {
	// Single-word descriptions must not be misclassified as collection headers
	input := `Configured Contexts
personal
/ (root)
Stuff`

	results := parseContexts(input)
	if len(results) != 1 {
		t.Fatalf("expected 1 context, got %d: %+v", len(results), results)
	}
	if results[0].Path != "/ (root)" || results[0].Text != "Stuff" {
		t.Errorf("context wrong: %+v", results[0])
	}
}

func TestParseContextsForCollection_Filters(t *testing.T) {
	// Multi-collection output: only the requested collection's contexts returned
	input := `Configured Contexts
sidekick
/ (root)
Work documentation and notes
learning
/ (root)
Learning notes and TIL
personal
/ (root)
Stuff`

	sidekick := parseContextsForCollection(input, "sidekick")
	if len(sidekick) != 1 || sidekick[0].Text != "Work documentation and notes" {
		t.Errorf("sidekick filter wrong: %+v", sidekick)
	}

	learning := parseContextsForCollection(input, "learning")
	if len(learning) != 1 || learning[0].Text != "Learning notes and TIL" {
		t.Errorf("learning filter wrong: %+v", learning)
	}

	personal := parseContextsForCollection(input, "personal")
	if len(personal) != 1 || personal[0].Text != "Stuff" {
		t.Errorf("personal filter wrong: %+v", personal)
	}

	all := parseContextsForCollection(input, "")
	if len(all) != 3 {
		t.Errorf("no-filter should return all 3, got %d: %+v", len(all), all)
	}
}
