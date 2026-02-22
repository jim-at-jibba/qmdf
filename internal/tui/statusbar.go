package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jamesguthriebest/qmdf/internal/qmd"
)

// renderStatusBar renders the single-line status bar at the bottom of the Search view.
func renderStatusBar(
	mode qmd.Mode,
	collection string,
	resultCount int,
	elapsedMs int64,
	loading bool,
	notification string,
	width int,
	showHelp bool,
	keyhint string,
) string {
	// Left: mode badge + optional collection name + result count
	badge := modeBadgeStyle.
		Foreground(lipgloss.Color("#1a1a2e")).
		Background(modeBadgeColor(mode)).
		Render(modeLabel(mode))

	collStr := ""
	if collection != "" {
		collStr = statusBarStyle.Render(" [" + collection + "]")
	}

	var countStr string
	if loading {
		countStr = statusBarStyle.Render(" searching…")
	} else if resultCount > 0 {
		elapsed := ""
		if elapsedMs > 0 {
			elapsed = fmt.Sprintf(" (%s)", formatElapsed(elapsedMs))
		}
		countStr = statusBarStyle.Render(fmt.Sprintf(" %d results%s", resultCount, elapsed))
	}

	left := badge + collStr + countStr

	// Right: notification or key hint
	var right string
	if notification != "" {
		right = notifyStyle.Render(notification)
	} else if !showHelp {
		right = helpStyle.Render(keyhint)
	}

	// Spacer to fill width
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	spacerWidth := width - leftWidth - rightWidth - 2
	if spacerWidth < 0 {
		spacerWidth = 0
	}

	spacer := strings.Repeat(" ", spacerWidth)

	return left + spacer + right
}
