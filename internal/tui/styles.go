package tui

import "github.com/charmbracelet/lipgloss"

// Color vars — concrete lipgloss.Color, NEVER AdaptiveColor (which sends OSC
// terminal queries that get captured as keyboard input while inside the TUI).
// Populated once by InitStyles before tea.NewProgram is called.
var (
	colorPrimary   lipgloss.Color
	colorSecondary lipgloss.Color
	colorMuted     lipgloss.Color
	colorSelected  lipgloss.Color
	colorBorder    lipgloss.Color
	colorError     lipgloss.Color
	colorSuccess   lipgloss.Color
	colorWarning   lipgloss.Color

	modeBadgeSearch  lipgloss.Color
	modeBadgeVSearch lipgloss.Color
	modeBadgeQuery   lipgloss.Color

	selectedBg lipgloss.Color
)

// Style vars — rebuilt by InitStyles.
var (
	selectedItemStyle  lipgloss.Style
	titleStyle         lipgloss.Style
	selectedTitleStyle lipgloss.Style
	pathStyle          lipgloss.Style
	selectedPathStyle  lipgloss.Style
	snippetStyle       lipgloss.Style
	paneStyle          lipgloss.Style
	activePaneStyle    lipgloss.Style
	inputPrefixStyle   lipgloss.Style
	statusBarStyle     lipgloss.Style
	modeBadgeStyle     lipgloss.Style
	helpStyle          lipgloss.Style
	notifyStyle        lipgloss.Style
	errorStyle         lipgloss.Style
)

func init() {
	// Safe default (dark) so styles are always non-zero even if InitStyles is
	// never called (e.g. in unit tests).
	InitStyles(true)
}

// InitStyles must be called once before tea.NewProgram, with the dark/light
// value pre-detected outside the TUI event loop.
func InitStyles(isDark bool) {
	pick := func(dark, light string) lipgloss.Color {
		if isDark {
			return lipgloss.Color(dark)
		}
		return lipgloss.Color(light)
	}

	colorPrimary = pick("#7dcfff", "#1a1a2e")
	colorSecondary = pick("#a9b1d6", "#374151")
	colorMuted = pick("#565f89", "#9ca3af")
	colorSelected = pick("#7aa2f7", "#1d4ed8")
	colorBorder = pick("#3b4261", "#d1d5db")
	colorError = pick("#f7768e", "#dc2626")
	colorSuccess = pick("#9ece6a", "#16a34a")
	colorWarning = pick("#e0af68", "#d97706")
	selectedBg = pick("#1e3a5f", "#dbeafe")

	modeBadgeSearch = pick("#7aa2f7", "#1d4ed8")
	modeBadgeVSearch = pick("#bb9af7", "#6d28d9")
	modeBadgeQuery = pick("#73daca", "#0f766e")

	buildStyles()
}

func buildStyles() {
	selectedItemStyle = lipgloss.NewStyle().
		Background(selectedBg).
		Foreground(colorSelected).
		Bold(true)

	titleStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	selectedTitleStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true)

	pathStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true)

	selectedPathStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Italic(true)

	snippetStyle = lipgloss.NewStyle().
		Foreground(colorSecondary)

	paneStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorBorder)

	activePaneStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(colorSelected)

	inputPrefixStyle = lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	modeBadgeStyle = lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	helpStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	notifyStyle = lipgloss.NewStyle().
		Foreground(colorSuccess)

	errorStyle = lipgloss.NewStyle().
		Foreground(colorError)
}

// _ prevents snippetStyle from being flagged as unused during linting.
var _ = snippetStyle
