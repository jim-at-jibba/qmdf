package editor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ClosedMsg is sent to the Bubble Tea program when the editor exits.
type ClosedMsg struct {
	Err error
}

// Open suspends the TUI and launches $EDITOR (or the given override) on the
// specified file. An optional line number can be provided (0 = no jump).
// Returns a tea.Cmd that will send ClosedMsg when the editor process exits.
func Open(filePath string, line int, editorOverride string) tea.Cmd {
	editor := editorOverride
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}

	// Resolve the binary (handles e.g. "code --wait" in EDITOR)
	parts := strings.Fields(editor)
	binary := parts[0]
	extraArgs := parts[1:]

	args := buildEditorArgs(binary, filePath, line)
	allArgs := append(extraArgs, args...) //nolint:gocritic

	cmd := exec.Command(binary, allArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ClosedMsg{Err: err}
	})
}

// OpenPager opens the file in the pager ($PAGER or less).
func OpenPager(filePath string) tea.Cmd {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less"
	}

	parts := strings.Fields(pager)
	binary := parts[0]
	extraArgs := parts[1:]

	args := append(extraArgs, filePath)
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return ClosedMsg{Err: err}
	})
}

// buildEditorArgs constructs the argument list for the given editor, optionally
// jumping to a specific line.
func buildEditorArgs(editor, filePath string, line int) []string {
	base := filepath.Base(editor)
	// Strip common suffixes (.exe, version suffixes like nvim.appimage, etc.)
	base = strings.ToLower(base)

	if line <= 0 {
		return []string{filePath}
	}

	switch {
	case strings.Contains(base, "vim") || strings.Contains(base, "vi"):
		// vim, nvim, gvim, etc.: +<line>
		return []string{fmt.Sprintf("+%d", line), filePath}
	case strings.Contains(base, "emacs"):
		return []string{fmt.Sprintf("+%d", line), filePath}
	case base == "code" || base == "code-insiders":
		// VS Code: --goto file:line
		return []string{"--goto", fmt.Sprintf("%s:%d", filePath, line)}
	case strings.Contains(base, "subl"):
		// Sublime Text: file:line
		return []string{fmt.Sprintf("%s:%d", filePath, line)}
	case strings.Contains(base, "hx") || strings.Contains(base, "helix"):
		// Helix: file:line
		return []string{fmt.Sprintf("%s:%d", filePath, line)}
	case strings.Contains(base, "zed"):
		return []string{fmt.Sprintf("%s:%d", filePath, line)}
	default:
		return []string{filePath}
	}
}
