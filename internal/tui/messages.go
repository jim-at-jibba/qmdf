package tui

import (
	"time"

	"github.com/jim-at-jibba/qmdf/internal/qmd"
)

// searchResultMsg carries results from a completed qmd search.
type searchResultMsg struct {
	requestID uint64
	results   []qmd.SearchResult
	elapsed   time.Duration
	err       error
}

// debounceTickMsg fires after the debounce delay and triggers a search.
type debounceTickMsg struct {
	requestID uint64
	query     string
	mode      qmd.Mode
}

// previewLoadedMsg carries the raw markdown content for a document.
type previewLoadedMsg struct {
	docID   string
	content string
	err     error
}

// editorClosedMsg is sent when the external editor process exits.
type editorClosedMsg struct {
	err error
}

// notificationMsg sets a transient status-bar notification.
type notificationMsg struct {
	text string
}
