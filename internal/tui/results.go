package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jim-at-jibba/qmdf/internal/qmd"
)

// renderResultList renders the results list for the left pane.
func renderResultList(results []qmd.SearchResult, selected, width, height int, loading bool, query string) string {
	if len(results) == 0 {
		return renderEmptyState(loading, query, width, height)
	}

	innerWidth := width - 4 // account for border + padding
	if innerWidth < 1 {
		innerWidth = 1
	}

	var sb strings.Builder
	linesUsed := 0
	maxLines := height - 2 // account for border

	for i, r := range results {
		if linesUsed >= maxLines {
			break
		}

		title := r.Title
		if title == "" {
			title = filepath.Base(r.DisplayPath())
		}

		// Truncate to fit
		title = truncate(title, innerWidth-2)
		path := truncate(r.DisplayPath(), innerWidth-2)

		if i == selected {
			titleLine := selectedTitleStyle.Render("▸ " + title)
			pathLine := selectedPathStyle.Render("  " + path)
			block := selectedItemStyle.Width(innerWidth).Render(titleLine + "\n" + pathLine)
			sb.WriteString(block)
		} else {
			titleLine := titleStyle.Render("  " + title)
			pathLine := pathStyle.Render("  " + path)
			sb.WriteString(titleLine + "\n" + pathLine)
		}

		linesUsed += 2
		if i < len(results)-1 && linesUsed < maxLines {
			sb.WriteByte('\n')
			linesUsed++
		}
	}

	return sb.String()
}

func renderEmptyState(loading bool, query string, width, height int) string {
	var msg string
	switch {
	case loading:
		msg = "Searching…"
	case query == "":
		msg = "Type to search"
	default:
		msg = "No results"
	}

	style := lipgloss.NewStyle().
		Width(width - 4).
		Height(height - 2).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(colorMuted)

	return style.Render(msg)
}

// scoreBar returns a small visual indicator for the score (0.0–1.0).
func scoreBar(score float64) string {
	pct := int(score * 5)
	if pct > 5 {
		pct = 5
	}
	filled := strings.Repeat("█", pct)
	empty := strings.Repeat("░", 5-pct)

	var color lipgloss.TerminalColor
	switch {
	case score >= 0.8:
		color = colorSuccess
	case score >= 0.5:
		color = colorWarning
	default:
		color = colorMuted
	}

	return lipgloss.NewStyle().Foreground(color).Render(filled + empty)
}

// truncate shortens s to at most n runes, appending "…" if cut.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}

// modeLabel returns a short display label for the mode.
func modeLabel(m qmd.Mode) string {
	switch m {
	case qmd.ModeVSearch:
		return "VSEARCH"
	case qmd.ModeQuery:
		return "QUERY"
	default:
		return "SEARCH"
	}
}

// modeBadgeColor returns the badge foreground color for the mode.
func modeBadgeColor(m qmd.Mode) lipgloss.TerminalColor {
	switch m {
	case qmd.ModeVSearch:
		return modeBadgeVSearch
	case qmd.ModeQuery:
		return modeBadgeQuery
	default:
		return modeBadgeSearch
	}
}

// cycleMode advances to the next search mode.
func cycleMode(current qmd.Mode) qmd.Mode {
	switch current {
	case qmd.ModeSearch:
		return qmd.ModeVSearch
	case qmd.ModeVSearch:
		return qmd.ModeQuery
	default:
		return qmd.ModeSearch
	}
}

// formatElapsed returns a human-readable elapsed time string.
func formatElapsed(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000)
}

// _ ensures scoreBar is used (it's available for future use)
var _ = scoreBar
