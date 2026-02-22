package qmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Client wraps the qmd CLI.
type Client struct {
	Collection string
	Results    int
	MinScore   float64
}

// searchTimeout returns the appropriate timeout for the given mode.
func searchTimeout(mode Mode) time.Duration {
	if mode == ModeQuery {
		return 30 * time.Second
	}
	return 10 * time.Second
}

// Search runs `qmd search --mode <mode> <query>` and returns results.
func (c *Client) Search(query string, mode Mode) ([]SearchResult, error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout(mode))
	defer cancel()

	args := c.buildSearchArgs(query, mode)
	cmd := exec.CommandContext(ctx, "qmd", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("qmd timed out after %s", searchTimeout(mode))
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("qmd exited %d: %s", exitErr.ExitCode(), stderr.String())
		}
		return nil, fmt.Errorf("qmd error: %w", err)
	}

	return ParseSearchResults(stdout.Bytes())
}

func (c *Client) buildSearchArgs(query string, mode Mode) []string {
	args := []string{"search"}

	if c.Collection != "" {
		args = append(args, "--collection", c.Collection)
	}
	if c.Results > 0 {
		args = append(args, "--results", strconv.Itoa(c.Results))
	}
	if c.MinScore > 0 {
		args = append(args, "--min-score", strconv.FormatFloat(c.MinScore, 'f', 2, 64))
	}

	// Mode flag varies by qmd version; try the most common form
	switch mode {
	case ModeVSearch:
		args = append(args, "--mode", "vsearch")
	case ModeQuery:
		args = append(args, "--mode", "query")
	}

	args = append(args, "--json")
	args = append(args, query)
	return args
}

// GetDocument fetches the full content of a document by its docID.
// qmd get expects the docID with or without a '#' prefix — we try both.
func (c *Client) GetDocument(docID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id := docID
	if !strings.HasPrefix(id, "#") {
		id = "#" + id
	}

	args := []string{"get", id}
	if c.Collection != "" {
		args = append(args, "--collection", c.Collection)
	}

	cmd := exec.CommandContext(ctx, "qmd", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// If that failed, retry without '#'
		cmd2 := exec.CommandContext(ctx, "qmd", append([]string{"get", docID}, args[2:]...)...)
		var stdout2 bytes.Buffer
		cmd2.Stdout = &stdout2
		if err2 := cmd2.Run(); err2 == nil {
			return stdout2.String(), nil
		}
		return "", fmt.Errorf("qmd get %s failed: %s", docID, stderr.String())
	}
	return stdout.String(), nil
}

// CheckInstalled returns an error if qmd is not found in PATH.
func CheckInstalled() error {
	_, err := exec.LookPath("qmd")
	if err != nil {
		return fmt.Errorf("qmd not found in PATH — install with: npm install -g @tobilu/qmd")
	}
	return nil
}
