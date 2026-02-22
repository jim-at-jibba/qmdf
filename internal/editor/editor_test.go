package editor

import "testing"

func TestGuiWaitFlag(t *testing.T) {
	cases := []struct {
		binary   string
		wantFlag string
	}{
		{"code", "--wait"},
		{"cursor", "--wait"},
		{"windsurf", "--wait"},
		{"zed", "--wait"},
		{"subl", "--wait"},
		{"vim", ""},
		{"nvim", ""},
		{"nano", ""},
		{"hx", ""},
	}
	for _, tc := range cases {
		got := guiWaitFlag(tc.binary)
		if got != tc.wantFlag {
			t.Errorf("guiWaitFlag(%q) = %q, want %q", tc.binary, got, tc.wantFlag)
		}
	}
}

func TestBuildEditorArgs_NoLine(t *testing.T) {
	args := buildEditorArgs("vim", "/path/to/file.md", 0)
	if len(args) != 1 || args[0] != "/path/to/file.md" {
		t.Errorf("expected [file], got %v", args)
	}
}

func TestBuildEditorArgs_Vim(t *testing.T) {
	args := buildEditorArgs("vim", "/path/to/file.md", 42)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %v", args)
	}
	if args[0] != "+42" {
		t.Errorf("expected '+42', got %q", args[0])
	}
	if args[1] != "/path/to/file.md" {
		t.Errorf("expected file path, got %q", args[1])
	}
}

func TestBuildEditorArgs_Nvim(t *testing.T) {
	args := buildEditorArgs("nvim", "/file.md", 10)
	if args[0] != "+10" {
		t.Errorf("expected '+10' for nvim, got %q", args[0])
	}
}

func TestBuildEditorArgs_VSCode(t *testing.T) {
	args := buildEditorArgs("code", "/file.md", 5)
	// Expects: ["--goto", "/file.md:5"]
	if len(args) != 2 {
		t.Fatalf("expected 2 args for code, got %v", args)
	}
	if args[0] != "--goto" {
		t.Errorf("expected '--goto' for code, got %q", args[0])
	}
	if args[1] != "/file.md:5" {
		t.Errorf("expected '/file.md:5', got %q", args[1])
	}
}

func TestBuildEditorArgs_Helix(t *testing.T) {
	args := buildEditorArgs("hx", "/file.md", 7)
	if len(args) != 1 || args[0] != "/file.md:7" {
		t.Errorf("expected ['/file.md:7'] for hx, got %v", args)
	}
}

func TestBuildEditorArgs_Sublime(t *testing.T) {
	args := buildEditorArgs("subl", "/file.md", 3)
	if args[0] != "/file.md:3" {
		t.Errorf("expected '/file.md:3' for subl, got %q", args[0])
	}
}

func TestBuildEditorArgs_UnknownEditor(t *testing.T) {
	args := buildEditorArgs("nano", "/file.md", 99)
	// Unknown editors just get the file path (no line jump)
	if len(args) != 1 || args[0] != "/file.md" {
		t.Errorf("expected ['/file.md'] for nano, got %v", args)
	}
}

func TestBuildEditorArgs_Emacs(t *testing.T) {
	args := buildEditorArgs("emacs", "/file.md", 20)
	if args[0] != "+20" {
		t.Errorf("expected '+20' for emacs, got %q", args[0])
	}
}
