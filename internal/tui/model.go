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

	// Input (search)
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

	// ── Collections view ──────────────────────────────────────────────────
	activeView ViewMode

	collections       []qmd.CollectionInfo
	collectionCursor  int
	collectionContexts []qmd.ContextInfo
	collectionLoading bool
	collectionErr     error
	collectionOutput  string

	// Multi-step input prompt
	inputMode         CollectionInputMode
	inputPrompt       string
	inputField        textinput.Model
	pendingAddPath    string
	pendingAddName    string
	pendingContextPath string
	pendingRenameName  string
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
		inputField:   newInputField(),
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
			cmds = append(cmds, m.fetchPreview(m.results[0]))
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
			cmds = append(cmds, sendNotification("Editor error: "+msg.Err.Error()))
		}

	// ── Internal notification ──────────────────────────────────────────────
	case notificationMsg:
		m.notification = msg.text
		m.notificationAt = time.Now()
		cmds = append(cmds, clearNotificationAfter(2*time.Second))

	case clearNotificationMsg:
		m.notification = ""

	// ── Collections loaded ─────────────────────────────────────────────────
	case collectionsLoadedMsg:
		m.collectionLoading = false
		if msg.err != nil {
			m.collectionErr = msg.err
		} else {
			m.collectionErr = nil
			m.collections = msg.collections
			if m.collectionCursor >= len(m.collections) {
				m.collectionCursor = max(0, len(m.collections)-1)
			}
		}

	// ── Collection action completed ────────────────────────────────────────
	case collectionActionMsg:
		m.collectionLoading = false
		if msg.err != nil {
			m.collectionErr = msg.err
			m.collectionOutput = msg.err.Error()
			cmds = append(cmds, sendNotification("Error: "+msg.err.Error()))
		} else {
			m.collectionErr = nil
			out := strings.TrimSpace(msg.output)
			if out == "" {
				out = msg.action + " completed"
			}
			m.collectionOutput = out
			cmds = append(cmds, sendNotification(msg.action+" done"))
			// Reload the collection list after any mutation
			cmds = append(cmds, m.loadCollections())
			// Reload contexts after context mutations
			if msg.action == "add-context" || msg.action == "remove-context" {
				cmds = append(cmds, m.loadContexts())
			}
			// After adding a new collection, automatically reindex so files are indexed.
			if msg.action == "add" {
				cmds = append(cmds, sendNotification("Collection added — reindexing…"))
				cmds = append(cmds, m.runCollectionAction("reindex", func() (string, error) {
					return m.client.Update(false)
				}))
			}
		}

	// ── Contexts loaded ────────────────────────────────────────────────────
	case contextsLoadedMsg:
		if msg.err == nil {
			m.collectionContexts = msg.contexts
		}

	// ── Keyboard input ─────────────────────────────────────────────────────
	case tea.KeyMsg:
		cmds = append(cmds, m.handleKey(msg)...)
	}

	return m, tea.Batch(cmds...)
}

// handleKey routes key events to global handlers then per-view handlers.
func (m *Model) handleKey(msg tea.KeyMsg) []tea.Cmd {
	// ctrl+c always quits
	if keyMatches(msg, m.keys.Quit) {
		return []tea.Cmd{tea.Quit}
	}

	// Backtick toggles views
	if keyMatches(msg, m.keys.ToggleView) {
		return m.toggleView()
	}

	// Help toggle is global
	if keyMatches(msg, m.keys.ToggleHelp) {
		m.showHelp = !m.showHelp
		m.help.ShowAll = m.showHelp
		return nil
	}

	// Delegate to the active view
	switch m.activeView {
	case ViewCollections:
		return m.handleCollectionKey(msg)
	default:
		return m.handleSearchKey(msg)
	}
}

// handleSearchKey processes key events in the Search view.
func (m *Model) handleSearchKey(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch {
	// ── Quit on Esc in search view ─────────────────────────────────────────
	case keyMatches(msg, m.keys.Esc):
		return []tea.Cmd{tea.Quit}

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
			realPath, err := qmd.ResolveFilePath(r.File)
			if err != nil {
				cmds = append(cmds, sendNotification("Cannot resolve path: "+err.Error()))
				break
			}
			if m.printMode {
				fmt.Println(realPath)
				return []tea.Cmd{tea.Quit}
			}
			cmds = append(cmds, editor.Open(realPath, 0, m.cfg.Editor))
		}

	// ── Pager ─────────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.Pager):
		if r := m.selectedResult(); r != nil {
			realPath, err := qmd.ResolveFilePath(r.File)
			if err != nil {
				cmds = append(cmds, sendNotification("Cannot resolve path: "+err.Error()))
				break
			}
			cmds = append(cmds, editor.OpenPager(realPath))
		}

	// ── Copy path ─────────────────────────────────────────────────────────
	case keyMatches(msg, m.keys.CopyPath):
		if r := m.selectedResult(); r != nil {
			realPath, err := qmd.ResolveFilePath(r.File)
			if err != nil {
				realPath = r.File // fall back to qmd:// URI
			}
			if err := clipboard.WriteAll(realPath); err == nil {
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

	tabBar := renderTabBar(m.activeView, m.width)

	switch m.activeView {
	case ViewCollections:
		body := renderCollectionsView(m)
		hint := renderCollectionsHintBar(m)
		return lipgloss.JoinVertical(lipgloss.Left, tabBar, body, hint)

	default: // ViewSearch
		// Build the input row
		prefix := inputPrefixStyle.Render("  ")
		inputRow := prefix + m.input.View()

		spinnerStr := ""
		if m.loading {
			spinnerStr = " " + m.spinner.View()
		}
		inputRow += spinnerStr

		// Body height: total – input(1) – tabBar(1) – border(2) – status(1)
		bodyHeight := m.height - 5
		if bodyHeight < 1 {
			bodyHeight = 1
		}

		if m.cfg.NoPreview {
			// Single-pane mode
			list := renderResultList(m.results, m.selected, m.width-2, bodyHeight, m.loading, m.lastQuery)
			leftPane := paneStyle.Width(m.width - 2).Height(bodyHeight).Render(list)
			status := renderStatusBar(m.mode, m.cfg.Collection, m.resultCount, m.elapsedMs, m.loading, m.notification, m.width, m.showHelp, m.shortHint())

			if m.showHelp {
				helpView := helpStyle.Render(m.help.View(m.keys))
				return lipgloss.JoinVertical(lipgloss.Left, inputRow, tabBar, leftPane, helpView, status)
			}
			return lipgloss.JoinVertical(lipgloss.Left, inputRow, tabBar, leftPane, status)
		}

		// Two-pane mode
		listW, previewW := m.paneSizes()

		list := renderResultList(m.results, m.selected, listW, bodyHeight, m.loading, m.lastQuery)
		leftPane := paneStyle.Width(listW - 2).Height(bodyHeight).Render(list)

		previewContent := m.viewport.View()
		rightPane := activePaneStyle.Width(previewW - 2).Height(bodyHeight).Render(previewContent)

		body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		status := renderStatusBar(m.mode, m.cfg.Collection, m.resultCount, m.elapsedMs, m.loading, m.notification, m.width, m.showHelp, m.shortHint())

		if m.showHelp {
			helpView := helpStyle.Render(m.help.View(m.keys))
			return lipgloss.JoinVertical(lipgloss.Left, inputRow, tabBar, body, helpView, status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, inputRow, tabBar, body, status)
	}
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
	if m.activeView == ViewCollections || m.cfg.NoPreview {
		return *m
	}
	_, previewW := m.paneSizes()
	bodyHeight := m.height - 5 // input + tabBar + border(2) + status
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
	parts := []string{"↑↓ nav", "enter open", "tab mode", "` collections", "? help", "^c quit"}
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
	return m.fetchPreview(*r)
}

func (m *Model) fetchPreview(r qmd.SearchResult) tea.Cmd {
	if r.DocID == "" {
		return nil
	}
	m.previewDocID = r.DocID

	// Cache hit (keyed by docID)
	if content, ok := m.previewCache.get(r.DocID); ok {
		m.setPreviewContent(content)
		return nil
	}

	// Async fetch via `qmd get #docid`
	client := m.client
	docID := r.DocID
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
