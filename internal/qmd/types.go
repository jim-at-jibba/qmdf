package qmd

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Mode represents the qmd search mode.
type Mode string

const (
	ModeSearch  Mode = "search"
	ModeVSearch Mode = "vsearch"
	ModeQuery   Mode = "query"
)

// SearchResult is a single document returned by qmd search.
// Field names match qmd's actual JSON output exactly.
type SearchResult struct {
	DocID   string  `json:"docid"`   // e.g. "#5ada41"
	File    string  `json:"file"`    // e.g. "qmd://collection/relative/path.md"
	Score   float64 `json:"score"`
	Title   string  `json:"title"`
	Context string  `json:"context"`
	Snippet string  `json:"snippet"`
}

// DisplayPath returns a short human-readable path by stripping the
// "qmd://collection/" prefix so the UI shows "knowledge-base/file.md"
// rather than the full URI.
func (r SearchResult) DisplayPath() string {
	// "qmd://sidekick/knowledge-base/file.md" → "knowledge-base/file.md"
	if !strings.HasPrefix(r.File, "qmd://") {
		return r.File
	}
	rest := strings.TrimPrefix(r.File, "qmd://")
	idx := strings.Index(rest, "/")
	if idx == -1 {
		return rest
	}
	return rest[idx+1:]
}

// ParseSearchResults parses qmd JSON output defensively.
// qmd outputs a bare JSON array; this also handles a {"results":[...]} wrapper
// for future-proofing.
func ParseSearchResults(data []byte) ([]SearchResult, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// Peek at the first non-whitespace byte to decide the shape.
	var firstByte byte
	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			firstByte = b
			break
		}
	}

	if firstByte == '{' {
		// Wrapped object format: {"results": [...]} or {"results": null}
		var wrapped struct {
			Results []SearchResult `json:"results"`
		}
		if err := json.Unmarshal(data, &wrapped); err != nil {
			return nil, fmt.Errorf("cannot parse qmd output: %w", err)
		}
		if wrapped.Results == nil {
			return []SearchResult{}, nil
		}
		return wrapped.Results, nil
	}

	// Bare array format (qmd's actual output)
	var results []SearchResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("cannot parse qmd output: %w", err)
	}
	return results, nil
}
