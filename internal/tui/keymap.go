package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for the TUI.
type KeyMap struct {
	Up         key.Binding
	Down       key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	Select     key.Binding
	OpenEditor key.Binding
	CopyPath   key.Binding
	CopyDocID  key.Binding
	CycleMode  key.Binding
	ToggleHelp key.Binding
	Pager      key.Binding
	Quit       key.Binding

	// Preview scroll
	PreviewUp   key.Binding
	PreviewDown key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "ctrl+k", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "ctrl+j", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "page down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open editor"),
		),
		OpenEditor: key.NewBinding(
			key.WithKeys("enter", "e"),
			key.WithHelp("e/enter", "open editor"),
		),
		CopyPath: key.NewBinding(
			key.WithKeys("ctrl+y"),
			key.WithHelp("^y", "copy path"),
		),
		CopyDocID: key.NewBinding(
			key.WithKeys("ctrl+i"),
			key.WithHelp("^i", "copy docid"),
		),
		CycleMode: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "cycle mode"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Pager: key.NewBinding(
			key.WithKeys("ctrl+p"),
			key.WithHelp("^p", "pager"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "esc"),
			key.WithHelp("^c/esc", "quit"),
		),
		PreviewUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("^u", "preview ↑"),
		),
		PreviewDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("^d", "preview ↓"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.OpenEditor, k.CycleMode, k.ToggleHelp, k.Quit}
}

// FullHelp implements help.KeyMap.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.OpenEditor, k.Pager, k.CopyPath, k.CopyDocID},
		{k.CycleMode, k.PreviewUp, k.PreviewDown},
		{k.ToggleHelp, k.Quit},
	}
}
