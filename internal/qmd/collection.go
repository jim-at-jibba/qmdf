package qmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// indexConfig mirrors the structure of ~/.config/qmd/index.yml.
type indexConfig struct {
	Collections map[string]collectionEntry `yaml:"collections"`
}

type collectionEntry struct {
	Path string `yaml:"path"`
}

var (
	collectionCacheMu sync.RWMutex
	collectionCache   map[string]string // collection name → root path
)

// ResolveFilePath converts a qmd:// URI to a real filesystem path.
//
//	"qmd://sidekick/knowledge-base/file.md"
//	→ "/Users/alice/notes/sidekick/knowledge-base/file.md"
//
// Returns the input unchanged if it isn't a qmd:// URI.
// Returns an error if the collection is unknown or the config can't be read.
func ResolveFilePath(qmdURI string) (string, error) {
	if !strings.HasPrefix(qmdURI, "qmd://") {
		return qmdURI, nil
	}

	rest := strings.TrimPrefix(qmdURI, "qmd://")
	slashIdx := strings.Index(rest, "/")
	if slashIdx == -1 {
		return "", fmt.Errorf("invalid qmd URI: %s", qmdURI)
	}

	collection := rest[:slashIdx]
	relativePath := rest[slashIdx+1:]

	roots, err := loadCollectionRoots()
	if err != nil {
		return "", fmt.Errorf("reading qmd config: %w", err)
	}

	root, ok := roots[collection]
	if !ok {
		return "", fmt.Errorf("collection %q not found in ~/.config/qmd/index.yml", collection)
	}

	return filepath.Join(root, relativePath), nil
}

// loadCollectionRoots reads ~/.config/qmd/index.yml and returns a map of
// collection name → absolute root path.  Results are cached in-process.
func loadCollectionRoots() (map[string]string, error) {
	collectionCacheMu.RLock()
	if collectionCache != nil {
		defer collectionCacheMu.RUnlock()
		return collectionCache, nil
	}
	collectionCacheMu.RUnlock()

	collectionCacheMu.Lock()
	defer collectionCacheMu.Unlock()

	// Double-checked locking
	if collectionCache != nil {
		return collectionCache, nil
	}

	path := qmdConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg indexConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	m := make(map[string]string, len(cfg.Collections))
	for name, entry := range cfg.Collections {
		m[name] = entry.Path
	}
	collectionCache = m
	return m, nil
}

// qmdConfigPath returns the path to qmd's index.yml config file.
func qmdConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "qmd", "index.yml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "qmd", "index.yml")
}
