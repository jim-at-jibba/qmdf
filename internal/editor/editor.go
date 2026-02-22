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

	// Resolve the binary (handles e.g. "code --wait" already in $EDITOR)
	parts := strings.Fields(editor)
	binary := parts[0]
	extraArgs := parts[1:]

	// GUI editors detach from the terminal and exit their CLI process immediately.
	// Inject --wait (or equivalent) so tea.ExecProcess blocks until the file is closed.
	// Only add it when it isn't already present (user may have set EDITOR="code --wait").
	if guiWaitFlag(binary) != "" && !containsArg(extraArgs, guiWaitFlag(binary)) {
		extraArgs = append(extraArgs, guiWaitFlag(binary))
	}

	args := buildEditorArgs(binary, filePath, line)
	allArgs := append(extraArgs, args...) //nolint:gocritic

	cmd := exec.Command(binary, allArgs...)

	// Open /dev/tty directly so the editor gets its own file descriptor to the
	// terminal. Using os.Stdin risks a race with bubbletea's internal stdin
	// reader, which causes nvim (and other terminal editors) to receive a
	// non-interactive stdin and exit immediately.
	tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if ttyErr == nil {
		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if tty != nil {
			tty.Close()
		}
		return ClosedMsg{Err: err}
	})
}

// guiWaitFlag returns the flag that makes a GUI editor's CLI block until the
// file is closed, or "" for terminal editors that already block naturally.
func guiWaitFlag(binary string) string {
	base := strings.ToLower(filepath.Base(binary))
	// Strip .exe on Windows
	base = strings.TrimSuffix(base, ".exe")
	switch base {
	case "code", "code-insiders", "codium":
		return "--wait"
	case "cursor":
		return "--wait"
	case "windsurf":
		return "--wait"
	case "subl", "sublime_text":
		return "--wait"
	case "zed":
		return "--wait"
	case "atom":
		return "--wait"
	case "mate": // TextMate
		return "--wait"
	}
	return "" // terminal editors (vim, nvim, nano, emacs, helix…) block naturally
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

	tty, ttyErr := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if ttyErr == nil {
		cmd.Stdin = tty
		cmd.Stdout = tty
		cmd.Stderr = tty
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if tty != nil {
			tty.Close()
		}
		return ClosedMsg{Err: err}
	})
}

// buildEditorArgs constructs the argument list for the given editor, optionally
// jumping to a specific line.
func buildEditorArgs(editor, filePath string, line int) []string {
	base := strings.ToLower(filepath.Base(editor))
	base = strings.TrimSuffix(base, ".exe")

	if line <= 0 {
		return []string{filePath}
	}

	switch {
	case strings.Contains(base, "vim") || base == "vi":
		// vim, nvim, gvim, etc.: +<line>
		return []string{fmt.Sprintf("+%d", line), filePath}
	case strings.Contains(base, "emacs"):
		return []string{fmt.Sprintf("+%d", line), filePath}
	case base == "code" || base == "code-insiders" || base == "codium" || base == "cursor" || base == "windsurf":
		// VS Code family: --goto file:line  (--wait is injected separately)
		return []string{"--goto", fmt.Sprintf("%s:%d", filePath, line)}
	case strings.Contains(base, "subl") || base == "sublime_text":
		// Sublime Text: file:line
		return []string{fmt.Sprintf("%s:%d", filePath, line)}
	case base == "hx" || strings.Contains(base, "helix"):
		return []string{fmt.Sprintf("%s:%d", filePath, line)}
	case base == "zed":
		return []string{fmt.Sprintf("%s:%d", filePath, line)}
	default:
		return []string{filePath}
	}
}

func containsArg(args []string, needle string) bool {
	for _, a := range args {
		if a == needle {
			return true
		}
	}
	return false
}
