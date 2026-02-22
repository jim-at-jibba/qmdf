package tui

import "github.com/jamesguthriebest/qmdf/internal/qmd"

// collectionsLoadedMsg is sent when the collection list fetch completes.
type collectionsLoadedMsg struct {
	collections []qmd.CollectionInfo
	err         error
}

// collectionActionMsg is sent when a collection mutation (add/remove/rename/update/embed) completes.
type collectionActionMsg struct {
	action string
	output string
	err    error
}

// contextsLoadedMsg is sent when the context list fetch completes.
type contextsLoadedMsg struct {
	contexts []qmd.ContextInfo
	err      error
}
