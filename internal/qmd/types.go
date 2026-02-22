package qmd

import (
	"encoding/json"
	"fmt"
)

// Mode represents the qmd search mode.
type Mode string

const (
	ModeSearch  Mode = "search"
	ModeVSearch Mode = "vsearch"
	ModeQuery   Mode = "query"
)

// SearchResult is a single document returned by qmd search.
type SearchResult struct {
	DocID    string  `json:"docid"`
	FilePath string  `json:"filepath"`
	Score    float64 `json:"score"`
	Title    string  `json:"title"`
	Snippet  string  `json:"snippet"`
}

// searchResponse is the wrapped JSON format: {"results": [...]}
type searchResponse struct {
	Results []SearchResult `json:"results"`
}

// ParseSearchResults parses qmd JSON output defensively.
// Accepts both {"results": [...]} and bare [...] formats.
// {"results": null} is treated as zero results (not an error).
func ParseSearchResults(data []byte) ([]SearchResult, error) {
	if len(data) == 0 {
		return nil, nil
	}

	// Peek at the first non-whitespace byte to decide the shape.
	trimmed := []byte{}
	for _, b := range data {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			trimmed = append(trimmed, b)
			break
		}
	}

	if len(trimmed) > 0 && trimmed[0] == '{' {
		// Wrapped object format: {"results": [...]} or {"results": null}
		var wrapped searchResponse
		if err := json.Unmarshal(data, &wrapped); err != nil {
			return nil, fmt.Errorf("cannot parse qmd output: %w", err)
		}
		// nil results == no results (not an error)
		if wrapped.Results == nil {
			return []SearchResult{}, nil
		}
		return wrapped.Results, nil
	}

	// Bare array format: [...]
	var results []SearchResult
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("cannot parse qmd output: %w", err)
	}
	return results, nil
}
