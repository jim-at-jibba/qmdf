package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

const previewCacheMax = 20

// previewCache is a simple FIFO-eviction cache keyed by docID.
type previewCache struct {
	order []string
	data  map[string]string
}

func newPreviewCache() *previewCache {
	return &previewCache{data: make(map[string]string)}
}

func (c *previewCache) get(key string) (string, bool) {
	v, ok := c.data[key]
	return v, ok
}

func (c *previewCache) set(key, value string) {
	if _, exists := c.data[key]; !exists {
		c.order = append(c.order, key)
		if len(c.order) > previewCacheMax {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.data, oldest)
		}
	}
	c.data[key] = value
}

// renderMarkdownWith renders raw markdown using the given glamour renderer.
// Falls back to plain text if the renderer is nil or rendering fails.
func renderMarkdownWith(content string, r *glamour.TermRenderer) string {
	if content == "" {
		return ""
	}
	if r == nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return out
}

// renderPreviewPane renders the right-hand preview pane content.
func renderPreviewPane(content string, width, height int, loading bool, docID string) string {
	innerWidth := width - 4
	if innerWidth < 1 {
		innerWidth = 1
	}
	innerHeight := height - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	if loading {
		style := lipgloss.NewStyle().
			Width(innerWidth).
			Height(innerHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorMuted)
		return style.Render("Loading preview…")
	}

	if content == "" {
		style := lipgloss.NewStyle().
			Width(innerWidth).
			Height(innerHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(colorMuted)
		if docID == "" {
			return style.Render("Select a result to preview")
		}
		return style.Render("No preview available")
	}

	lines := strings.Split(content, "\n")
	for len(lines) < innerHeight {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
