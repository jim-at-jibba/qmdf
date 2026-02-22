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

	// Tab bar
	tabActiveStyle   lipgloss.Style
	tabInactiveStyle lipgloss.Style

	// Collection view
	collItemStyle     lipgloss.Style
	collSelectedStyle lipgloss.Style
	collDetailKeyStyle lipgloss.Style
	collDetailValStyle lipgloss.Style
	collPromptStyle    lipgloss.Style
	collHintStyle      lipgloss.Style
	collLogTitleStyle  lipgloss.Style
)

func init() {
	// Safe default so styles are always non-zero even if InitStyles is
	// never called (e.g. in unit tests).
	InitStyles()
}

// InitStyles must be called once before tea.NewProgram.
func InitStyles() {
	colorPrimary = lipgloss.Color("4")   // Blue
	colorSecondary = lipgloss.Color("7") // White
	colorMuted = lipgloss.Color("8")     // Bright Black (gray)
	colorSelected = lipgloss.Color("12") // Bright Blue
	colorBorder = lipgloss.Color("8")    // Bright Black
	colorError = lipgloss.Color("1")     // Red
	colorSuccess = lipgloss.Color("2")   // Green
	colorWarning = lipgloss.Color("3")   // Yellow
	selectedBg = lipgloss.Color("0")     // Black (bg highlight)

	modeBadgeSearch = lipgloss.Color("4")  // Blue
	modeBadgeVSearch = lipgloss.Color("5") // Magenta
	modeBadgeQuery = lipgloss.Color("6")   // Cyan

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

	tabActiveStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true).
		Padding(0, 1)

	tabInactiveStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Padding(0, 1)

	collItemStyle = lipgloss.NewStyle().
		Foreground(colorPrimary)

	collSelectedStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true)

	collDetailKeyStyle = lipgloss.NewStyle().
		Foreground(colorMuted).
		Width(10)

	collDetailValStyle = lipgloss.NewStyle().
		Foreground(colorSecondary)

	collPromptStyle = lipgloss.NewStyle().
		Foreground(colorWarning).
		Bold(true)

	collHintStyle = lipgloss.NewStyle().
		Foreground(colorMuted)

	collLogTitleStyle = lipgloss.NewStyle().
		Foreground(colorSelected).
		Bold(true)
}

// _ prevents snippetStyle from being flagged as unused during linting.
var _ = snippetStyle
