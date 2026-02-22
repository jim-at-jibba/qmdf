package qmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const collectionTimeout = 30 * time.Second
const longOpTimeout = 120 * time.Second

// CollectionInfo holds metadata about a qmd collection.
type CollectionInfo struct {
	Name      string
	Pattern   string
	FileCount int
	Updated   string
	Path      string
}

// ContextInfo holds a single qmd context entry.
type ContextInfo struct {
	Path string
	Text string
}

// runQmd executes qmd with the given args and returns stdout output.
// stderr is captured and included in error messages.
func runQmd(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "qmd", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("qmd timed out after %s", timeout)
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			errMsg := strings.TrimSpace(stderr.String())
			if errMsg == "" {
				errMsg = strings.TrimSpace(stdout.String())
			}
			return "", fmt.Errorf("qmd exited %d: %s", exitErr.ExitCode(), errMsg)
		}
		return "", fmt.Errorf("qmd error: %w", err)
	}
	return stdout.String(), nil
}

// ListCollections runs `qmd collection list` and returns parsed collection info.
func (c *Client) ListCollections() ([]CollectionInfo, error) {
	out, err := runQmd(collectionTimeout, "collection", "list")
	if err != nil {
		return nil, err
	}
	return parseCollections(out), nil
}

// parseCollections parses the text output of `qmd collection list`.
//
// Handles multiple formats qmd might use:
//
//	Configured Collections        ← section header (skipped)
//	sidekick                      ← collection name
//	  Pattern: **/*.md            ← field (indented or not)
//	  Files: 34
//	  Updated: 4d ago
//
// Also handles inline parens format:
//
//	sidekick (34 files, **/*.md, updated 4d ago)
//
// The key rule: any line containing ":" is a field or header, never a name.
// Multi-word lines without ":" are headers. Single-word lines without ":" are names.
func parseCollections(text string) []CollectionInfo {
	var results []CollectionInfo
	var current *CollectionInfo

	reFiles := regexp.MustCompile(`(\d+)`)
	// "name (metadata...)" inline format
	reInline := regexp.MustCompile(`^(\S+)\s+\((.+)\)\s*$`)
	reUpdatedInline := regexp.MustCompile(`updated\s+(\S+(?:\s+ago)?)`)
	reMaskInline := regexp.MustCompile(`\*[\w.*]+`)

	for _, rawLine := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}

		// Inline format: "name (N files, mask, updated X ago)"
		if m := reInline.FindStringSubmatch(trimmed); m != nil {
			if current != nil {
				results = append(results, *current)
			}
			info := CollectionInfo{Name: m[1]}
			inner := m[2]
			if fm := reFiles.FindString(inner); fm != "" {
				info.FileCount, _ = strconv.Atoi(fm)
			}
			if um := reUpdatedInline.FindStringSubmatch(inner); um != nil {
				info.Updated = um[1]
			}
			if mm := reMaskInline.FindString(inner); mm != "" {
				info.Pattern = mm
			}
			current = &info
			continue
		}

		// Any line with a colon is a "key: value" field or section header — never a name.
		if colonIdx := strings.Index(trimmed, ":"); colonIdx >= 0 {
			if current != nil {
				key := strings.ToLower(strings.TrimSpace(trimmed[:colonIdx]))
				val := strings.TrimSpace(trimmed[colonIdx+1:])
				switch key {
				case "pattern", "mask", "glob":
					current.Pattern = val
				case "files", "count", "file count", "indexed":
					if fm := reFiles.FindString(val); fm != "" {
						current.FileCount, _ = strconv.Atoi(fm)
					}
				case "updated", "last updated":
					current.Updated = val
				case "path", "directory", "location":
					current.Path = val
				}
			}
			continue
		}

		// Multi-word lines without ":" are section headers — skip.
		if strings.ContainsAny(trimmed, " \t") {
			continue
		}

		// Single token without a colon — collection name.
		// Must start with a letter, digit, or underscore.
		first := trimmed[0]
		if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || (first >= '0' && first <= '9') || first == '_') {
			continue
		}

		if current != nil {
			results = append(results, *current)
		}
		current = &CollectionInfo{Name: trimmed}
	}

	if current != nil {
		results = append(results, *current)
	}

	return results
}

// expandHome replaces a leading ~ with the user's home directory.
// exec.Command does not go through a shell, so ~ is never expanded otherwise.
func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return home + path[1:]
		}
	}
	return path
}

// AddCollection runs `qmd collection add <path> [--name <name>] [--mask <mask>]`.
func (c *Client) AddCollection(path, name, mask string) (string, error) {
	args := []string{"collection", "add", expandHome(path)}
	if name != "" {
		args = append(args, "--name", name)
	}
	if mask != "" {
		args = append(args, "--mask", mask)
	}
	return runQmd(collectionTimeout, args...)
}

// RemoveCollection runs `qmd collection remove <name>`.
func (c *Client) RemoveCollection(name string) (string, error) {
	return runQmd(collectionTimeout, "collection", "remove", name)
}

// RenameCollection runs `qmd collection rename <old> <new>`.
func (c *Client) RenameCollection(oldName, newName string) (string, error) {
	return runQmd(collectionTimeout, "collection", "rename", oldName, newName)
}

// Update runs `qmd update [--pull]` (long-running: 120s timeout).
func (c *Client) Update(pull bool) (string, error) {
	args := []string{"update"}
	if pull {
		args = append(args, "--pull")
	}
	return runQmd(longOpTimeout, args...)
}

// Embed runs `qmd embed [-f]` (long-running: 120s timeout).
func (c *Client) Embed(force bool) (string, error) {
	args := []string{"embed"}
	if force {
		args = append(args, "-f")
	}
	return runQmd(longOpTimeout, args...)
}

// ListContexts runs `qmd context list` and returns parsed context info.
func (c *Client) ListContexts() ([]ContextInfo, error) {
	out, err := runQmd(collectionTimeout, "context", "list")
	if err != nil {
		return nil, err
	}
	return parseContexts(out), nil
}

// parseContexts parses the text output of `qmd context list`.
//
// qmd uses a two-line format per entry:
//
//	Configured Contexts    ← section header (skipped)
//	sidekick               ← collection name (skipped)
//	/ (root)               ← context path
//	Work documentation and notes  ← description for the path above
//	/projects
//	Code and project files
//
// Also handles the inline "path: description" format as a fallback.
func parseContexts(text string) []ContextInfo {
	var results []ContextInfo
	var pendingPath string

	flush := func() {
		if pendingPath != "" {
			results = append(results, ContextInfo{Path: pendingPath})
			pendingPath = ""
		}
	}

	isPathLine := func(s string) bool {
		return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "~") || strings.HasPrefix(s, "./")
	}

	for _, rawLine := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(rawLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Inline "path: description" on one line
		if idx := strings.Index(trimmed, ": "); idx > 0 && isPathLine(trimmed) {
			flush()
			results = append(results, ContextInfo{
				Path: strings.TrimSpace(trimmed[:idx]),
				Text: strings.TrimSpace(trimmed[idx+2:]),
			})
			continue
		}

		// Section/scope headers: end with ":" or are single-word non-path tokens
		if strings.HasSuffix(trimmed, ":") {
			flush()
			continue
		}
		if !strings.ContainsAny(trimmed, " \t") && !isPathLine(trimmed) {
			// Single word, not a path — collection name header
			flush()
			continue
		}

		// Path line (starts with /, ~, ./)
		if isPathLine(trimmed) {
			flush()
			pendingPath = trimmed
			continue
		}

		// Description line following a path
		if pendingPath != "" {
			results = append(results, ContextInfo{Path: pendingPath, Text: trimmed})
			pendingPath = ""
			continue
		}

		// Multi-word line with no pending path — section header, skip
	}

	flush()
	return results
}

// AddContext runs `qmd context add [path] "text"`.
func (c *Client) AddContext(path, text string) (string, error) {
	args := []string{"context", "add"}
	if path != "" {
		args = append(args, path)
	}
	args = append(args, text)
	return runQmd(collectionTimeout, args...)
}

// RemoveContext runs `qmd context rm <path>`.
func (c *Client) RemoveContext(path string) (string, error) {
	return runQmd(collectionTimeout, "context", "rm", path)
}

// GetStatus runs `qmd status` and returns the raw output.
func (c *Client) GetStatus() (string, error) {
	return runQmd(collectionTimeout, "status")
}
