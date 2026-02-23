# qmdf

[![CI](https://github.com/jim-at-jibba/qmdf/actions/workflows/ci.yml/badge.svg)](https://github.com/jim-at-jibba/qmdf/actions/workflows/ci.yml)
[![Release](https://github.com/jim-at-jibba/qmdf/actions/workflows/release.yml/badge.svg)](https://github.com/jim-at-jibba/qmdf/releases)

A fast, keyboard-driven TUI for searching and managing [qmd](https://github.com/tobilu/qmd) Markdown collections. Wraps the `qmd` CLI with an interactive two-pane interface: live search results on the left, rendered Markdown preview on the right.

![qmdf demo](https://github.com/user-attachments/assets/886a6546-4895-4f66-bcbc-086044c6b212)

- Single static binary, no runtime dependencies
- Sub-10ms startup
- Three search modes: full-text, vector/semantic, LLM-reranked
- Full collection management: add, remove, rename, reindex, embed, context CRUD
- Shell integration via `--print` mode

---

## Prerequisites

- **[qmd](https://github.com/tobilu/qmd)** — the underlying document index engine
- **Go 1.25+** — only required to build from source

---

## Installation

### Pre-built binaries (recommended)

Download the latest release for your platform from the [releases page](https://github.com/jim-at-jibba/qmdf/releases).

### go install

```bash
go install github.com/jim-at-jibba/qmdf@latest
```

### Build from source

```bash
git clone https://github.com/jim-at-jibba/qmdf.git
cd qmdf
make build          # produces ./qmdf
make install        # installs to $GOPATH/bin
```

### Cross-compile

```bash
make cross
```

Produces binaries in `dist/` for Linux (amd64/arm64), macOS (Intel/Apple Silicon), and Windows (amd64).

---

## Quick Start

```bash
# Search the default collection
qmdf

# Search a specific collection
qmdf --collection my-notes

# Use vector search mode
qmdf --mode vsearch

# Single-pane (no preview)
qmdf --no-preview
```

---

## Views

qmdf has two views. Press `` ` `` to switch between them.

### Search View

```
  Search term…
╭──────────────────────────╮ ╭──────────────────────────────────╮
│ ▸ My First Document      │ │ # My First Document              │
│   Another Note           │ │                                  │
│   Project Overview       │ │ This is the rendered preview of  │
│   …                      │ │ the selected Markdown document.  │
╰──────────────────────────╯ ╰──────────────────────────────────╯
  [search]  3 results  12ms
```

**Key bindings:**

| Key | Action |
|-----|--------|
| Type anything | Search (150ms debounce) |
| `↑` / `k` / `ctrl+k` | Move up |
| `↓` / `j` / `ctrl+j` | Move down |
| `pgup` / `pgdn` | Page up/down (5 at a time) |
| `tab` | Cycle search mode (search → vsearch → query) |
| `enter` / `e` | Open in `$EDITOR` |
| `ctrl+p` | Open in `$PAGER` |
| `ctrl+u` / `ctrl+d` | Scroll preview up/down |
| `ctrl+y` | Copy file path to clipboard |
| `ctrl+i` | Copy docID to clipboard |
| `` ` `` | Switch to Collections view |
| `?` | Toggle help overlay |
| `ctrl+c` / `esc` | Quit |

**Search modes:**

| Mode | Description | Shortcut |
|------|-------------|---------|
| `search` | Full-text BM25 | `tab` |
| `vsearch` | Vector/semantic search | `tab` |
| `query` | LLM-reranked results | `tab` |

### Collections View

```
  Search    ▸ Collections                               ` switch
╭────────────────────╮ ╭────────────────────────────────╮
│ ▸ my-notes (124)   │ │ my-notes                       │
│   sidekick (34)    │ │ ────────────────────────────── │
│   work (89)        │ │ Files    124                   │
│                    │ │ Pattern  **/*.md                │
│                    │ │ Updated  2h ago                 │
│                    │ │                                 │
│                    │ │ Contexts                        │
│                    │ │ ────────────────────────────── │
│                    │ │   / (root)                      │
│                    │ │     Personal knowledge base     │
╰────────────────────╯ ╰────────────────────────────────╯
                       ╭────────────────────────────────╮
                       │ Output ────────────────────    │
                       │ reindex completed               │
                       ╰────────────────────────────────╯
  a:add  d:del  r:rename  u:reindex  e:embed  c:ctx  x:rmctx
```

**Key bindings:**

| Key | Action |
|-----|--------|
| `↑` / `k` / `ctrl+k` | Move up |
| `↓` / `j` / `ctrl+j` | Move down |
| `a` | Add collection |
| `d` | Delete collection (confirms by name) |
| `r` | Rename collection |
| `u` | Reindex (`qmd update`) |
| `U` | Reindex + pull (`qmd update --pull`) |
| `e` | Generate embeddings (`qmd embed`) |
| `E` | Force-regenerate embeddings (`qmd embed -f`) |
| `c` | Add context to selected collection |
| `x` | Remove context from selected collection |
| `enter` | Select collection and switch to Search |
| `esc` | Back to Search view |

**Adding a collection** is a three-step prompt:
1. Filesystem path (e.g. `~/notes` — tilde is expanded automatically)
2. Name (leave empty to auto-derive from the path)
3. File mask (e.g. `**/*.md` — leave empty for the qmd default)

After creation, qmdf automatically runs `qmd update` to index the files.

**Contexts** are descriptions scoped to a collection that qmd uses for LLM-aware query expansion. They are stored under `qmd://<collection>` and shown in the detail pane.

---

## Configuration

Config file: `~/.config/qmdf/config.yaml`

```yaml
# qmd collection to search (default: none — searches all)
collection: my-notes

# Default search mode: search | vsearch | query
mode: search

# Maximum number of results shown
results: 10

# Minimum relevance score filter (0.0–1.0)
min_score: 0.0

# Disable the preview pane
no_preview: false

# Preview pane width as a fraction of terminal width (0.0–1.0)
preview_width: 0.55

# Override $EDITOR (e.g. for GUI editors)
editor: "code --wait"

# Supports env var prefixes for custom configs (e.g. Neovim app names)
# editor: "NVIM_APPNAME=kick nvim"
```

### Environment variables

All options are also available as environment variables with the `QMDF_` prefix:

```bash
QMDF_COLLECTION=my-notes qmdf
QMDF_MODE=vsearch qmdf
QMDF_NO_PREVIEW=true qmdf
```

**Priority order:** CLI flags > environment variables > config file > defaults

---

## CLI Flags

```
-c, --collection <name>     qmd collection name
-m, --mode <mode>           search|vsearch|query
-n, --results <int>         max results
    --min-score <float>     minimum relevance score
    --no-preview            single-pane mode (no preview)
    --print                 print selected path to stdout (shell integration)
    --no-qmd-check          skip qmd installation warning
```

---

## Shell Integration

Use `--print` to pipe the selected file path into other commands:

```bash
# Open selected file in $EDITOR from a shell script
$EDITOR "$(qmdf --print)"

# cd to the directory of the selected file
cd "$(dirname "$(qmdf --print)")"

# Copy selected file to a destination
cp "$(qmdf --print)" ~/backup/
```

In `--print` mode the TUI runs normally. Pressing `enter` prints the resolved filesystem path to stdout and exits.

---

## Editor

qmdf opens files using `$EDITOR` (falling back to `$VISUAL`, then `nano`). You can override it in the config file with the `editor` key.

The editor string supports:
- Simple binaries: `nvim`, `vim`, `nano`
- GUI editors with flags: `code --wait`, `zed --wait`
- **Environment variable prefixes**: `NVIM_APPNAME=kick nvim` — useful for custom Neovim configs

```yaml
# ~/.config/qmdf/config.yaml
editor: "NVIM_APPNAME=kick nvim"
```

If `$EDITOR` is set to an absolute path that no longer exists (e.g. after reinstalling), qmdf automatically looks up the binary name on `$PATH`.

---

## Build Targets

```
make build      Compile ./qmdf
make install    Install to $GOPATH/bin
make test       Run all tests
make lint       Run go vet
make tidy       Tidy go.mod
make cross      Cross-compile for all platforms into dist/
make clean      Remove build artefacts
```

## Releasing

Releases are automated via GitHub Actions and [GoReleaser](https://goreleaser.com). Tagging a commit publishes pre-built binaries for all platforms to the [releases page](https://github.com/jim-at-jibba/qmdf/releases).

```bash
git tag v1.0.0
git push origin v1.0.0
```

GoReleaser builds Linux (amd64/arm64), macOS (amd64/arm64), and Windows (amd64) archives and generates a `checksums.txt`.

---

## Architecture

```
main.go
cmd/root.go                  CLI flags, initialise styles, start tea.Program
internal/
  qmd/
    types.go                 SearchResult, ParseSearchResults
    client.go                Search(), GetDocument(), CheckInstalled()
    collections.go           Collection/context client methods + text parsers
  tui/
    model.go                 Root tea.Model, debounce, view routing
    collections.go           Collections view: rendering, input handling, async cmds
    keymap.go                Key binding definitions
    styles.go                lipgloss styles (no AdaptiveColor)
    results.go               Result list rendering, cycleMode()
    preview.go               previewCache (FIFO, 20 entries), glamour rendering
    statusbar.go             Status bar
    messages.go              tea.Msg types
    collections_messages.go  Collections-specific tea.Msg types
  editor/
    editor.go                tea.ExecProcess for $EDITOR / $PAGER, line-jump args
  config/
    config.go                Viper config, defaults
```

Key design notes:

- **Debounce** — 150 ms via `tea.Tick` + monotonic `requestID`; stale responses are discarded
- **Preview cache** — FIFO eviction, 20 entries, keyed by docID
- **No `lipgloss.AdaptiveColor`** — it sends an OSC 11 terminal query that leaks into keyboard input inside the TUI; dark/light detection is done once before `tea.NewProgram`
- **Editor handoff** — `tea.ExecProcess` suspends the TUI and restores it when the editor exits
- **Long-running commands** (`update`, `embed`) — 120 s timeout, spinner shown in the Output pane while running

---

## License

MIT
