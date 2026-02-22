package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jim-at-jibba/qmdf/internal/qmd"
)

// infoBoxContentHeight is the number of content lines in the bottom-right info pane
// shown in the Search view.
const infoBoxContentHeight = 3

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

// renderSearchInfoBox renders the bottom-right info pane in the Search view.
// It mirrors the Output pane in the Collections view.
func renderSearchInfoBox(m Model, width int) string {
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}

	// Title row: "qmdf ──────── v<version>"
	label := collLogTitleStyle.Render("qmdf")
	ver := collHintStyle.Render("v" + m.version)
	labelW := lipgloss.Width(label)
	verW := lipgloss.Width(ver)
	sepLen := innerW - labelW - verW - 2
	if sepLen < 0 {
		sepLen = 0
	}
	titleRow := label + " " + collHintStyle.Render(strings.Repeat("─", sepLen)) + " " + ver

	// Line 1: collection
	collVal := m.cfg.Collection
	if collVal == "" {
		collVal = "all"
	}
	line1 := infoLabelStyle.Render("collection") + " " + collDetailValStyle.Render(truncate(collVal, innerW-12))

	// Line 2: mode
	line2 := infoLabelStyle.Render("mode") + " " + collDetailValStyle.Render(string(m.mode))

	return titleRow + "\n" + line1 + "\n" + line2
}
