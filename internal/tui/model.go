package tui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/jamesguthriebest/qmdf/internal/config"
	"github.com/jamesguthriebest/qmdf/internal/editor"
	"github.com/jamesguthriebest/qmdf/internal/qmd"
)

// globalRequestID is a monotonic counter for stale-response detection.
var globalRequestID uint64

func nextRequestID() uint64 {
	return atomic.AddUint64(&globalRequestID, 1)
}

const debounceDuration = 150 * time.Millisecond

// Model is the root Bubble Tea model.
type Model struct {
	cfg    *config.Config
	client *qmd.Client
	keys   KeyMap
	help   help.Model

	// Input
	input textinput.Model

	// State
	mode     qmd.Mode
	loading  bool
	spinner  spinner.Model
	err      error
	showHelp bool

	// Results
	results     []qmd.SearchResult
	selected    int
	resultCount int

	// Preview
	viewport        viewport.Model
	previewCache    *previewCache
	previewDocID    string
	previewReady    bool
	glamourRenderer *glamour.TermRenderer // cached; rebuilt only on width change
	glamourWidth    int
	isDark          bool

	// Debounce
	requestID uint64
	lastQuery string

	// Timing
	searchStart time.Time
	elapsedMs   int64

	// Notification
	notification   string
	notificationAt time.Time

	// Layout
	width  int
	height int

	// Modes
	printMode bool // output path to stdout on Enter, then quit
}

// New creates a new Model with the given config.
// isDark must be pre-detected before tea.NewProgram is called (see cmd/root.go).
func New(cfg *config.Config, isDark bool) Model {
	ti := textinput.New()
	ti.Placeholder = "Search documents…"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorSelected) // uses pre-built style var

	mode := qmd.Mode(cfg.Mode)
	if mode != qmd.ModeSearch && mode != qmd.ModeVSearch && mode != qmd.ModeQuery {
		mode = qmd.ModeSearch
	}

	return Model{
		cfg:          cfg,
		client:       &qmd.Client{Collection: cfg.Collection, Results: cfg.Results, MinScore: cfg.MinScore},
		keys:         DefaultKeyMap(),
		help:         help.New(),
		input:        ti,
		mode:         mode,
		spinner:      s,
		previewCache: newPreviewCache(),
		printMode:    cfg.PrintMode,
		isDark:       isDark,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ── Window resize ──────────────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.recalculateLayout()

	// ── Spinner tick ───────────────────────────────────────────────────────
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	// ── Debounce tick: fire search ─────────────────────────────────────────
	case debounceTickMsg:
		if msg.requestID != m.requestID {
			break // stale — a newer keystroke superseded this
		}
		cmds = append(cmds, m.doSearch(msg.query, msg.mode))

	// ── Search results returned ────────────────────────────────────────────
	case searchResultMsg:
		if msg.requestID != m.requestID {
			break // stale response
		}
		m.loading = false
		m.elapsedMs = msg.elapsed.Milliseconds()
		if msg.err != nil {
			m.err = msg.err
			m.results = nil
			m.resultCount = 0
			break
		}
		m.err = nil
		m.results = msg.results
		m.resultCount = len(msg.results)
		m.selected = 0

		if len(m.results) > 0 {
			cmds = append(cmds, m.fetchPreview(m.results[0].DocID))
		} else {
			m.previewDocID = ""
			m.previewReady = false
			m.viewport.SetContent("")
		}

	// ── Preview loaded ─────────────────────────────────────────────────────
	case previewLoadedMsg:
		if msg.err != nil {
			// Show error in preview pane but don't kill the app
			m.viewport.SetContent(errorStyle.Render("Preview error: " + msg.err.Error()))
			m.previewReady = true
			break
		}
		m.previewCache.set(msg.docID, msg.content)
		// Only display if this is still the selected document
		if msg.docID == m.previewDocID {
			m.setPreviewContent(msg.content)
		}

	// ── Editor / pager closed ──────────────────────────────────────────────
	case editor.ClosedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		}

	// ── Internal notification ──────────────────────────────────────────────
	case notificationMsg:
		m.notification = msg.text
		m.notificationAt = time.Now()
		cmds = append(cmds, clearNotificationAfter(2*time.Second))

	case clearNotificationMsg:
		m.notification = ""

	// ── Keyboard input ─────────────────────────────────────────────────────
	case tea.KeyMsg:
		cmds = append(cmds, m.handleKey(msg)...)
	}

	return m, tea.Batch(cmds...)
}

// handleKey processes a key event and returns any resulting commands.
func (m *Model) handleKey(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch {
	// ── Quit ──────────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.Quit):
		return []tea.Cmd{tea.Quit}

	// ── Help toggle ───────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.ToggleHelp):
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp

	// ── Navigation ────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.Up):
		if m.selected > 0 {
			m.selected--
			cmds = append(cmds, m.fetchPreviewForSelected())
		}

	case keyMatches(msg, m.keys.Down):
		if m.selected < len(m.results)-1 {
			m.selected++
			cmds = append(cmds, m.fetchPreviewForSelected())
		}

	case keyMatches(msg, m.keys.PageUp):
		m.selected -= 5
		if m.selected < 0 {
			m.selected = 0
		}
		cmds = append(cmds, m.fetchPreviewForSelected())

	case keyMatches(msg, m.keys.PageDown):
		m.selected += 5
		if m.selected >= len(m.results) {
			m.selected = len(m.results) - 1
		}
		cmds = append(cmds, m.fetchPreviewForSelected())

	// ── Preview scroll ────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.PreviewUp):
		m.viewport.HalfViewUp()

	case keyMatches(msg, m.keys.PreviewDown):
		m.viewport.HalfViewDown()

	// ── Mode cycle ────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.CycleMode):
		m.mode = cycleMode(m.mode)
		// Re-run search immediately with new mode
		if m.lastQuery != "" {
			cmds = append(cmds, m.scheduleDebouncedSearch(m.lastQuery, 0))
		}

	// ── Open editor ───────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.Select):
		if r := m.selectedResult(); r != nil {
			if m.printMode {
				fmt.Println(r.FilePath)
				return []tea.Cmd{tea.Quit}
			}
			cmds = append(cmds, editor.Open(r.FilePath, 0, m.cfg.Editor))
		}

	// ── Pager ─────────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.Pager):
		if r := m.selectedResult(); r != nil {
			cmds = append(cmds, editor.OpenPager(r.FilePath))
		}

	// ── Copy path ─────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.CopyPath):
		if r := m.selectedResult(); r != nil {
			if err := clipboard.WriteAll(r.FilePath); err == nil {
				cmds = append(cmds, sendNotification("Copied path"))
			} else {
				cmds = append(cmds, sendNotification("Clipboard error"))
			}
		}

	// ── Copy docID ────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.CopyDocID):
		if r := m.selectedResult(); r != nil {
			if err := clipboard.WriteAll(r.DocID); err == nil {
				cmds = append(cmds, sendNotification("Copied docid"))
			} else {
				cmds = append(cmds, sendNotification("Clipboard error"))
			}
		}

	// ── Text input (anything else goes to the search box) ─────────────────
	default:
		var tiCmd tea.Cmd
		m.input, tiCmd = m.input.Update(msg)
		cmds = append(cmds, tiCmd)

		newQuery := m.input.Value()
		if newQuery != m.lastQuery {
			m.lastQuery = newQuery
			cmds = append(cmds, m.scheduleDebouncedSearch(newQuery, debounceDuration))
		}
	}

	return cmds
}

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "Initialising…"
	}

	// Build the input row
	prefix := inputPrefixStyle.Render("  ")
	inputRow := prefix + m.input.View()

	spinnerStr := ""
	if m.loading {
		spinnerStr = " " + m.spinner.View()
	}
	inputRow += spinnerStr

	// Body height: total – input(1) – border(2) – status(1)
	bodyHeight := m.height - 4
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	if m.cfg.NoPreview {
		// Single-pane mode
		list := renderResultList(m.results, m.selected, m.width-2, bodyHeight, m.loading, m.lastQuery)
		leftPane := paneStyle.Width(m.width - 2).Height(bodyHeight).Render(list)
		status := renderStatusBar(m.mode, m.resultCount, m.elapsedMs, m.loading, m.notification, m.width, m.showHelp, m.shortHint())

		if m.showHelp {
			helpView := helpStyle.Render(m.help.View(m.keys))
			return lipgloss.JoinVertical(lipgloss.Left, inputRow, leftPane, helpView, status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, inputRow, leftPane, status)
	}

	// Two-pane mode
	listW, previewW := m.paneSizes()

	list := renderResultList(m.results, m.selected, listW, bodyHeight, m.loading, m.lastQuery)
	leftPane := paneStyle.Width(listW - 2).Height(bodyHeight).Render(list)

	previewContent := m.viewport.View()
	rightPane := activePaneStyle.Width(previewW - 2).Height(bodyHeight).Render(previewContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	status := renderStatusBar(m.mode, m.resultCount, m.elapsedMs, m.loading, m.notification, m.width, m.showHelp, m.shortHint())

	if m.showHelp {
		helpView := helpStyle.Render(m.help.View(m.keys))
		return lipgloss.JoinVertical(lipgloss.Left, inputRow, body, helpView, status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, inputRow, body, status)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (m Model) selectedResult() *qmd.SearchResult {
	if len(m.results) == 0 || m.selected < 0 || m.selected >= len(m.results) {
		return nil
	}
	r := m.results[m.selected]
	return &r
}

func (m *Model) recalculateLayout() Model {
	if m.cfg.NoPreview {
		return *m
	}
	_, previewW := m.paneSizes()
	bodyHeight := m.height - 4
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	m.viewport = viewport.New(previewW-4, bodyHeight-2)
	if content, ok := m.previewCache.get(m.previewDocID); ok {
		m.viewport.SetContent(m.renderMarkdown(content, previewW-4))
	}
	return *m
}

// renderMarkdown renders markdown using a cached glamour renderer.
// The renderer is only recreated when the wrap width changes.
func (m *Model) renderMarkdown(content string, width int) string {
	if content == "" || width < 4 {
		return content
	}
	if m.glamourRenderer == nil || m.glamourWidth != width {
		style := "dark"
		if !m.isDark {
			style = "light"
		}
		r, err := glamour.NewTermRenderer(
			glamour.WithStandardStyle(style),
			glamour.WithWordWrap(width),
		)
		if err != nil {
			return content
		}
		m.glamourRenderer = r
		m.glamourWidth = width
	}
	return renderMarkdownWith(content, m.glamourRenderer)
}

func (m Model) paneSizes() (listW, previewW int) {
	frac := m.cfg.PreviewWidth
	if frac <= 0 || frac >= 1 {
		frac = 0.55
	}
	previewW = int(float64(m.width) * frac)
	listW = m.width - previewW
	if listW < 20 {
		listW = 20
	}
	if previewW < 20 {
		previewW = 20
	}
	return
}

func (m Model) shortHint() string {
	parts := []string{"↑↓ nav", "enter open", "tab mode", "? help", "^c quit"}
	return helpStyle.Render(strings.Join(parts, "  "))
}

func (m *Model) setPreviewContent(raw string) {
	_, previewW := m.paneSizes()
	rendered := m.renderMarkdown(raw, previewW-4)
	m.viewport.SetContent(rendered)
	m.viewport.GotoTop()
	m.previewReady = true
}

func (m *Model) fetchPreviewForSelected() tea.Cmd {
	r := m.selectedResult()
	if r == nil {
		return nil
	}
	return m.fetchPreview(r.DocID)
}

func (m *Model) fetchPreview(docID string) tea.Cmd {
	if docID == "" {
		return nil
	}
	m.previewDocID = docID

	// Cache hit
	if content, ok := m.previewCache.get(docID); ok {
		m.setPreviewContent(content)
		return nil
	}

	// Async fetch
	client := m.client
	return func() tea.Msg {
		content, err := client.GetDocument(docID)
		return previewLoadedMsg{docID: docID, content: content, err: err}
	}
}

// scheduleDebouncedSearch schedules a search after a delay.
// Pass delay=0 to fire immediately (still goes through debounce path for stale detection).
func (m *Model) scheduleDebouncedSearch(query string, delay time.Duration) tea.Cmd {
	id := nextRequestID()
	m.requestID = id
	m.loading = query != ""

	if delay == 0 {
		return func() tea.Msg {
			return debounceTickMsg{requestID: id, query: query, mode: m.mode}
		}
	}

	return tea.Tick(delay, func(t time.Time) tea.Msg {
		return debounceTickMsg{requestID: id, query: query, mode: m.mode}
	})
}

// doSearch runs the actual qmd search in a goroutine.
func (m *Model) doSearch(query string, mode qmd.Mode) tea.Cmd {
	if query == "" {
		m.loading = false
		m.results = nil
		m.resultCount = 0
		return nil
	}

	id := m.requestID
	client := m.client
	start := time.Now()

	return func() tea.Msg {
		results, err := client.Search(query, mode)
		return searchResultMsg{
			requestID: id,
			results:   results,
			elapsed:   time.Since(start),
			err:       err,
		}
	}
}

// ── Notification helpers ──────────────────────────────────────────────────

type clearNotificationMsg struct{}

func sendNotification(text string) tea.Cmd {
	return func() tea.Msg { return notificationMsg{text: text} }
}

func clearNotificationAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

// keyMatches delegates to bubbles/key.Matches, which also respects Enabled().
func keyMatches(msg tea.KeyMsg, b key.Binding) bool {
	return key.Matches(msg, b)
}
