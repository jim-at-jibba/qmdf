# CLAUDE.md — qmdf

## Project

`qmdf` is a Go/Bubble Tea TUI that wraps the [`qmd`](https://github.com/tobilu/qmd) CLI tool for searching Markdown document collections. It is a single static binary with zero runtime dependencies and <10ms startup.

## Build Commands

```bash
make build          # compile ./qmdf
make test           # go test ./...
make lint           # go vet ./...
make install        # go install to $GOPATH/bin
make cross          # cross-compile for linux/darwin/windows into dist/
```

Or directly:

```bash
go build -o qmdf .
go test ./...
GOOS=linux go build ./...   # cross-compile check
```

## Architecture

```
main.go                     # entry point → cmd.Execute()
cmd/root.go                 # cobra command, flag parsing, starts tea.Program
internal/
  qmd/
    types.go                # SearchResult, ParseSearchResults (handles both JSON formats)
    client.go               # Search(), GetDocument(), CheckInstalled()
  tui/
    model.go                # root tea.Model: Init/Update/View, debounce, key handling
    keymap.go               # key.Binding definitions (bubbles/key)
    styles.go               # lipgloss styles and colors
    messages.go             # custom tea.Msg types
    results.go              # result list rendering, helpers (truncate, cycleMode, etc.)
    preview.go              # glamour rendering, previewCache (LRU, 20 entries)
    statusbar.go            # status bar rendering
  editor/
    editor.go               # tea.ExecProcess for $EDITOR and $PAGER, line-jump args
  config/
    config.go               # viper config from ~/.config/qmdf/config.yaml
```

## Key Design Decisions

**Debounce**: 150ms via `tea.Tick` + monotonic `requestID`. Stale search results and ticks are discarded by comparing `requestID`.

**Query mode timeout**: 30 seconds (LLM reranking is slow). Other modes: 10 seconds.

**Editor handoff**: `tea.ExecProcess` suspends the TUI and gives the editor full terminal control. The TUI resumes on exit.

**Preview cache**: `previewCache` in `internal/tui/preview.go` — simple FIFO eviction, max 20 entries.

**JSON parsing**: `ParseSearchResults` in `internal/qmd/types.go` accepts both `{"results":[...]}` and `[...]` formats defensively.

**Two-pane layout**: `lipgloss.JoinHorizontal`, configurable split via `PreviewWidth` (default 0.55). Single-pane mode via `--no-preview`.

## qmd Integration Notes

- qmd is invoked as a subprocess: `qmd search --json [flags] <query>`
- DocIDs in qmd output are like `a1b2c3` (no `#` prefix); `qmd get` may require `#a1b2c3` — the client tries both.
- qmd JSON output is `{"results": [...]}` — bare array `[...]` is also handled as a fallback.

## Config File

`~/.config/qmdf/config.yaml`:

```yaml
collection: my-notes      # qmd collection name
mode: search              # search | vsearch | query
results: 10               # max results
min_score: 0.0            # minimum relevance score
no_preview: false
preview_width: 0.55       # fraction of terminal width for preview pane
editor: ""                # override $EDITOR (e.g. "code --wait")
```

Environment variables: `QMDF_COLLECTION`, `QMDF_MODE`, etc. (prefix `QMDF_`).

## Key Bindings (defaults)

| Key | Action |
|-----|--------|
| `↑` / `k` / `ctrl+k` | Move up |
| `↓` / `j` / `ctrl+j` | Move down |
| `pgup` / `pgdn` | Page navigation |
| `enter` / `e` | Open in $EDITOR |
| `ctrl+p` | Open in $PAGER |
| `ctrl+y` | Copy file path to clipboard |
| `ctrl+i` | Copy docID to clipboard |
| `tab` | Cycle search mode (search → vsearch → query) |
| `ctrl+u` / `ctrl+d` | Scroll preview up/down |
| `?` | Toggle help overlay |
| `ctrl+c` / `esc` | Quit |

## Shell Integration (--print mode)

```bash
# Open selected file in $EDITOR via shell
$EDITOR "$(qmdf --print)"

# cd to the directory of the selected file
cd "$(dirname "$(qmdf --print)")"
```

## Adding a New Search Mode

1. Add the mode constant to `internal/qmd/types.go`
2. Add the `--mode` arg in `buildSearchArgs` in `client.go`
3. Add the cycle step in `cycleMode` in `results.go`
4. Add the badge color in `modeBadgeColor` in `results.go`

## Running Without qmd Installed

Use `--no-qmd-check` to suppress the "qmd not found" warning. For dev/testing with canned output, create a mock script:

```bash
#!/usr/bin/env bash
# ~/bin/qmd (mock)
echo '{"results":[{"docid":"abc123","filepath":"/tmp/test.md","score":0.9,"title":"Test Doc","snippet":"hello world"}]}'
```
