package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jim-at-jibba/qmdf/internal/qmd"
)

// ViewMode selects between the Search and Collections views.
type ViewMode int

const (
	ViewSearch      ViewMode = iota
	ViewCollections ViewMode = iota
)

// CollectionInputMode tracks which multi-step input prompt is active.
type CollectionInputMode int

const (
	InputNone           CollectionInputMode = iota
	InputAddPath        CollectionInputMode = iota // step 1 of add: path
	InputAddName        CollectionInputMode = iota // step 2 of add: name
	InputAddMask        CollectionInputMode = iota // step 3 of add: file mask
	InputRenameTo       CollectionInputMode = iota // new name for rename
	InputConfirmDelete  CollectionInputMode = iota // type name to confirm delete
	InputContextText CollectionInputMode = iota // add-context description
)

// ── Tab bar ────────────────────────────────────────────────────────────────

// renderTabBar renders the two-tab navigation bar.
func renderTabBar(active ViewMode, width int) string {
	searchLabel := "Search"
	collLabel := "Collections"

	var searchTab, collTab string
	if active == ViewSearch {
		searchTab = tabActiveStyle.Render("▸ " + searchLabel)
		collTab = tabInactiveStyle.Render("  " + collLabel)
	} else {
		searchTab = tabInactiveStyle.Render("  " + searchLabel)
		collTab = tabActiveStyle.Render("▸ " + collLabel)
	}

	tabs := searchTab + "  " + collTab
	hint := tabInactiveStyle.Render("` switch")

	tabsW := lipgloss.Width(tabs)
	hintW := lipgloss.Width(hint)
	spacer := width - tabsW - hintW - 2
	if spacer < 0 {
		spacer = 0
	}

	return tabs + strings.Repeat(" ", spacer) + hint
}

// ── Collections view rendering ─────────────────────────────────────────────

// logPaneContentHeight is the number of content lines in the bottom-right log pane.
const logPaneContentHeight = 3

// renderCollectionsView renders the full collections body (list + detail panes).
// The right column is split: detail pane on top, log pane on bottom-right.
func renderCollectionsView(m Model) string {
	bodyHeight := m.height - 4 // tabBar(1) + border(2) + hintBar(1)
	if bodyHeight < logPaneContentHeight+7 {
		bodyHeight = logPaneContentHeight + 7
	}

	// Log pane: logPaneContentHeight content lines + 2 border = logPaneContentHeight+2 total rows.
	// Detail pane: bodyHeight - (logPaneContentHeight+2) content lines + 2 border.
	// Left pane spans full bodyHeight content height.
	detailContentH := bodyHeight - (logPaneContentHeight + 2)
	if detailContentH < 2 {
		detailContentH = 2
	}

	listW := m.width / 2
	detailW := m.width - listW

	listContent := renderCollectionList(m.collections, m.collectionCursor, listW, bodyHeight, m.collectionLoading, m.collectionErr)
	leftPane := paneStyle.Width(listW - 2).Height(bodyHeight).Render(listContent)

	detailContent := renderCollectionDetail(m, detailW, detailContentH)
	detailPane := paneStyle.Width(detailW - 2).Height(detailContentH).Render(detailContent)

	logContent := renderCollectionLog(m, detailW)
	logPane := paneStyle.Width(detailW - 2).Height(logPaneContentHeight).Render(logContent)

	rightCol := lipgloss.JoinVertical(lipgloss.Left, detailPane, logPane)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightCol)
}

// renderCollectionList renders the left pane list of collections.
func renderCollectionList(colls []qmd.CollectionInfo, cursor, width, height int, loading bool, err error) string {
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2
	if innerH < 1 {
		innerH = 1
	}

	if loading && len(colls) == 0 {
		return centerText("Loading collections…", innerW, innerH)
	}
	if err != nil && len(colls) == 0 {
		return centerText(errorStyle.Render("Error: "+err.Error()), innerW, innerH)
	}
	if len(colls) == 0 {
		return centerText(collHintStyle.Render("No collections — press a to add"), innerW, innerH)
	}

	var sb strings.Builder
	linesUsed := 0

	for i, c := range colls {
		if linesUsed >= innerH {
			break
		}

		nameStr := truncate(c.Name, innerW-3)
		suffix := ""
		if c.FileCount > 0 {
			suffix = collHintStyle.Render(fmt.Sprintf(" (%d)", c.FileCount))
		}

		var line string
		if i == cursor {
			line = collSelectedStyle.Render("▸ "+nameStr) + suffix
		} else {
			line = collItemStyle.Render("  "+nameStr) + suffix
		}
		sb.WriteString(line)
		linesUsed++

		if i < len(colls)-1 && linesUsed < innerH {
			sb.WriteByte('\n')
			linesUsed++
		}
	}

	return sb.String()
}

// renderCollectionDetail renders the right pane detail for the selected collection.
func renderCollectionDetail(m Model, width, height int) string {
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}

	if len(m.collections) == 0 {
		return centerText(collHintStyle.Render("Select a collection"), innerW, height-2)
	}

	if m.collectionCursor < 0 || m.collectionCursor >= len(m.collections) {
		return ""
	}

	c := m.collections[m.collectionCursor]
	var sb strings.Builder

	// Section heading
	heading := collSelectedStyle.Render(c.Name)
	sb.WriteString(heading + "\n")
	sb.WriteString(collHintStyle.Render(strings.Repeat("─", min(innerW, 30))) + "\n")

	writeField := func(label, val string) {
		if val == "" {
			return
		}
		l := collDetailKeyStyle.Render(label)
		v := collDetailValStyle.Render(truncate(val, innerW-lipgloss.Width(l)-1))
		sb.WriteString(l + " " + v + "\n")
	}

	if c.FileCount > 0 {
		writeField("Files  ", fmt.Sprintf("%d", c.FileCount))
	}
	if c.Pattern != "" {
		writeField("Pattern", c.Pattern)
	}
	if c.Updated != "" {
		writeField("Updated", c.Updated)
	}
	if c.Path != "" {
		writeField("Path   ", truncate(c.Path, innerW-10))
	}

	// Contexts section
	sb.WriteString("\n")
	sb.WriteString(collDetailKeyStyle.Render("Contexts") + "\n")
	sb.WriteString(collHintStyle.Render(strings.Repeat("─", min(innerW, 30))) + "\n")

	if len(m.collectionContexts) == 0 {
		sb.WriteString(collHintStyle.Render("(none — press c to add)") + "\n")
	} else {
		for _, ctx := range m.collectionContexts {
			pathStr := truncate(ctx.Path, innerW-2)
			sb.WriteString(collDetailValStyle.Render("  "+pathStr) + "\n")
			if ctx.Text != "" {
				textStr := truncate(ctx.Text, innerW-4)
				sb.WriteString(collHintStyle.Render("    "+textStr) + "\n")
			}
		}
	}

	return sb.String()
}

// renderCollectionLog renders the bottom-right log pane.
// Shows a spinner while an operation is running, or the last command's output.
func renderCollectionLog(m Model, width int) string {
	innerW := width - 4
	if innerW < 1 {
		innerW = 1
	}

	// Title row: "Output ─────────────────"
	label := collLogTitleStyle.Render("Output")
	labelW := lipgloss.Width(label)
	sepLen := innerW - labelW - 1
	if sepLen < 0 {
		sepLen = 0
	}
	titleRow := label + " " + collHintStyle.Render(strings.Repeat("─", sepLen))

	if m.collectionLoading {
		return titleRow + "\n" + collHintStyle.Render(m.spinner.View()+" running…")
	}
	if m.collectionOutput == "" {
		return titleRow + "\n" + collHintStyle.Render("─")
	}

	lines := strings.Split(strings.TrimSpace(m.collectionOutput), "\n")
	// logPaneContentHeight includes the title row, so content lines = height - 1
	maxContent := logPaneContentHeight - 1
	if maxContent < 1 {
		maxContent = 1
	}
	if len(lines) > maxContent {
		lines = lines[len(lines)-maxContent:]
	}

	lineStyle := collDetailValStyle
	if m.collectionErr != nil {
		lineStyle = errorStyle
	}

	var sb strings.Builder
	sb.WriteString(titleRow + "\n")
	for i, line := range lines {
		sb.WriteString(lineStyle.Render(truncate(line, innerW)))
		if i < len(lines)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// renderCollectionsHintBar renders the bottom hint/input row for the collections view.
func renderCollectionsHintBar(m Model) string {
	if m.inputMode != InputNone {
		prompt := collPromptStyle.Render(m.inputPrompt + " ")
		field := m.inputField.View()
		return prompt + field
	}

	if m.collectionLoading {
		return collHintStyle.Render(m.spinner.View() + " running…")
	}

	hints := []string{"a:add", "d:del", "r:rename", "u:reindex", "e:embed", "c:ctx", "x:rmctx", "↵:select", "esc:back"}
	return collHintStyle.Render(strings.Join(hints, "  "))
}

// ── Key handling ───────────────────────────────────────────────────────────

// handleCollectionKey processes key events in the Collections view.
func (m *Model) handleCollectionKey(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// When an input prompt is active, route to input handler
	if m.inputMode != InputNone {
		switch {
		case keyMatches(msg, m.keys.Esc):
			m.cancelCollectionInput()
		case msg.Type == tea.KeyEnter:
			cmds = append(cmds, m.handleCollectionInputSubmit()...)
		default:
			var cmd tea.Cmd
			m.inputField, cmd = m.inputField.Update(msg)
			cmds = append(cmds, cmd)
		}
		return cmds
	}

	// Normal navigation/action mode
	switch {
	case keyMatches(msg, m.keys.Esc):
		// Back to search view
		m.activeView = ViewSearch
		m.input.Focus()

	case keyMatches(msg, m.keys.CollUp):
		if m.collectionCursor > 0 {
			m.collectionCursor--
			m.collectionContexts = nil
			cmds = append(cmds, m.loadContexts())
		}

	case keyMatches(msg, m.keys.CollDown):
		if m.collectionCursor < len(m.collections)-1 {
			m.collectionCursor++
			m.collectionContexts = nil
			cmds = append(cmds, m.loadContexts())
		}

	case keyMatches(msg, m.keys.PageUp):
		m.collectionCursor -= 5
		if m.collectionCursor < 0 {
			m.collectionCursor = 0
		}
		m.collectionContexts = nil
		cmds = append(cmds, m.loadContexts())

	case keyMatches(msg, m.keys.PageDown):
		m.collectionCursor += 5
		if m.collectionCursor >= len(m.collections) {
			m.collectionCursor = len(m.collections) - 1
		}
		m.collectionContexts = nil
		cmds = append(cmds, m.loadContexts())

	case keyMatches(msg, m.keys.CollAdd):
		m.startCollectionInput(InputAddPath, "Collection path:")

	case keyMatches(msg, m.keys.CollDelete):
		if sel := m.selectedCollection(); sel != nil {
			m.pendingRenameName = sel.Name
			m.startCollectionInput(InputConfirmDelete,
				fmt.Sprintf("Type '%s' to confirm delete:", sel.Name))
		}

	case keyMatches(msg, m.keys.CollRename):
		if sel := m.selectedCollection(); sel != nil {
			m.pendingRenameName = sel.Name
			m.startCollectionInput(InputRenameTo,
				fmt.Sprintf("Rename '%s' to:", sel.Name))
		}

	case keyMatches(msg, m.keys.CollReindex):
		cmds = append(cmds, m.runCollectionAction("reindex", func() (string, error) {
			return m.client.Update(false)
		}))

	case keyMatches(msg, m.keys.CollReindexPull):
		cmds = append(cmds, m.runCollectionAction("reindex+pull", func() (string, error) {
			return m.client.Update(true)
		}))

	case keyMatches(msg, m.keys.CollEmbed):
		cmds = append(cmds, m.runCollectionAction("embed", func() (string, error) {
			return m.client.Embed(false)
		}))

	case keyMatches(msg, m.keys.CollEmbedForce):
		cmds = append(cmds, m.runCollectionAction("embed-force", func() (string, error) {
			return m.client.Embed(true)
		}))

	case keyMatches(msg, m.keys.CollContext):
		if sel := m.selectedCollection(); sel != nil {
			m.pendingContextPath = "qmd://" + sel.Name
			m.startCollectionInput(InputContextText, fmt.Sprintf("Context description for %s:", sel.Name))
		}

	case keyMatches(msg, m.keys.CollContextRm):
		if sel := m.selectedCollection(); sel != nil {
			path := "qmd://" + sel.Name
			cmds = append(cmds, m.runCollectionAction("remove-context", func() (string, error) {
				return m.client.RemoveContext(path)
			}))
		}

	case msg.Type == tea.KeyEnter:
		// Switch to Search view with the selected collection active
		if sel := m.selectedCollection(); sel != nil {
			m.client.Collection = sel.Name
			m.cfg.Collection = sel.Name
			cmds = append(cmds, sendNotification("Collection: "+sel.Name))
		}
		m.activeView = ViewSearch
		m.input.Focus()
	}

	return cmds
}

// handleCollectionInputSubmit processes Enter in an active input prompt.
func (m *Model) handleCollectionInputSubmit() []tea.Cmd {
	value := strings.TrimSpace(m.inputField.Value())

	switch m.inputMode {
	case InputAddPath:
		m.pendingAddPath = value
		m.startCollectionInput(InputAddName, "Collection name (leave empty for auto):")

	case InputAddName:
		m.pendingAddName = value
		m.startCollectionInput(InputAddMask, "File mask (e.g. **/*.md, leave empty for default):")

	case InputAddMask:
		path := m.pendingAddPath
		name := m.pendingAddName
		mask := value
		m.pendingAddPath = ""
		m.pendingAddName = ""
		m.cancelCollectionInput()
		if path == "" {
			return []tea.Cmd{sendNotification("Path is required")}
		}
		return []tea.Cmd{m.runCollectionAction("add", func() (string, error) {
			return m.client.AddCollection(path, name, mask)
		})}

	case InputRenameTo:
		newName := value
		oldName := m.pendingRenameName
		m.pendingRenameName = ""
		m.cancelCollectionInput()
		if newName == "" {
			return nil
		}
		return []tea.Cmd{m.runCollectionAction("rename", func() (string, error) {
			return m.client.RenameCollection(oldName, newName)
		})}

	case InputConfirmDelete:
		sel := m.selectedCollection()
		name := m.pendingRenameName
		m.pendingRenameName = ""
		m.cancelCollectionInput()
		if sel != nil && value == name {
			return []tea.Cmd{m.runCollectionAction("delete", func() (string, error) {
				return m.client.RemoveCollection(name)
			})}
		}
		// Wrong name typed — silently cancel

	case InputContextText:
		path := m.pendingContextPath
		text := value
		m.pendingContextPath = ""
		m.cancelCollectionInput()
		if text == "" {
			return nil
		}
		return []tea.Cmd{m.runCollectionAction("add-context", func() (string, error) {
			return m.client.AddContext(path, text)
		})}

	}

	return nil
}

// ── Input prompt helpers ───────────────────────────────────────────────────

func (m *Model) startCollectionInput(mode CollectionInputMode, prompt string) {
	m.inputMode = mode
	m.inputPrompt = prompt
	m.inputField.SetValue("")
	m.inputField.Focus()
}

func (m *Model) cancelCollectionInput() {
	m.inputMode = InputNone
	m.inputPrompt = ""
	m.inputField.SetValue("")
	m.inputField.Blur()
}

// ── Async command helpers ──────────────────────────────────────────────────

// loadCollections loads the collection list asynchronously.
func (m *Model) loadCollections() tea.Cmd {
	m.collectionLoading = true
	m.collectionErr = nil
	client := m.client
	return func() tea.Msg {
		collections, err := client.ListCollections()
		return collectionsLoadedMsg{collections: collections, err: err}
	}
}

// loadContexts loads the contexts for the currently selected collection.
func (m *Model) loadContexts() tea.Cmd {
	client := m.client
	collName := ""
	if sel := m.selectedCollection(); sel != nil {
		collName = sel.Name
	}
	return func() tea.Msg {
		contexts, err := client.ListContextsForCollection(collName)
		return contextsLoadedMsg{contexts: contexts, err: err}
	}
}

// runCollectionAction runs a collection mutation and returns a tea.Cmd.
// collectionOutput is intentionally NOT cleared here — it stays visible while
// the operation runs, replaced only when the new output arrives.
func (m *Model) runCollectionAction(action string, fn func() (string, error)) tea.Cmd {
	m.collectionLoading = true
	m.collectionErr = nil
	return func() tea.Msg {
		output, err := fn()
		return collectionActionMsg{action: action, output: output, err: err}
	}
}

// toggleView switches between Search and Collections views.
func (m *Model) toggleView() []tea.Cmd {
	if m.activeView == ViewCollections {
		m.activeView = ViewSearch
		m.input.Focus()
		return nil
	}
	m.activeView = ViewCollections
	m.input.Blur()
	return []tea.Cmd{m.loadCollections(), m.loadContexts(), m.spinner.Tick}
}

// selectedCollection returns a pointer to the currently selected CollectionInfo, or nil.
func (m *Model) selectedCollection() *qmd.CollectionInfo {
	if len(m.collections) == 0 {
		return nil
	}
	if m.collectionCursor < 0 || m.collectionCursor >= len(m.collections) {
		return nil
	}
	c := m.collections[m.collectionCursor]
	return &c
}

// newInputField creates a fresh text input for collection prompts.
func newInputField() textinput.Model {
	ti := textinput.New()
	ti.CharLimit = 512
	ti.Width = 50
	return ti
}

// centerText renders text centered in a box of the given inner dimensions.
func centerText(text string, width, height int) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(colorMuted).
		Render(text)
}
