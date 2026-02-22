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
		return 60 * time.Second // LLM reranking can be slow
	}
	if mode == ModeVSearch {
		return 30 * time.Second // vector search with HyDE expansion
	}
	return 10 * time.Second
}

// Search runs `qmd <mode> <query> --json [flags]` and returns results.
// mode is a subcommand: "search", "vsearch", or "query".
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
	cmd.Stderr = &stderr // progress lines go to stderr; capture for error messages only

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

// buildSearchArgs constructs the qmd argument list.
// qmd uses separate subcommands for each mode:
//
//	qmd search  <query> [flags]   — full-text BM25
//	qmd vsearch <query> [flags]   — vector similarity
//	qmd query   <query> [flags]   — LLM query expansion + reranking
func (c *Client) buildSearchArgs(query string, mode Mode) []string {
	// First arg is the subcommand, not a flag
	subcmd := string(mode)
	if subcmd == "" {
		subcmd = "search"
	}
	args := []string{subcmd}

	if c.Collection != "" {
		args = append(args, "-c", c.Collection)
	}
	if c.Results > 0 {
		args = append(args, "-n", strconv.Itoa(c.Results))
	}
	if c.MinScore > 0 {
		args = append(args, "--min-score", strconv.FormatFloat(c.MinScore, 'f', 2, 64))
	}

	args = append(args, "--json")
	args = append(args, query)
	return args
}

// GetDocument fetches the full markdown content of a document by its docID.
// The docID must include the '#' prefix (e.g. "#5ada41") as returned by search.
func (c *Client) GetDocument(docID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ensure # prefix
	id := docID
	if !strings.HasPrefix(id, "#") {
		id = "#" + id
	}

	args := []string{"get", id}
	if c.Collection != "" {
		args = append(args, "-c", c.Collection)
	}

	cmd := exec.CommandContext(ctx, "qmd", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("qmd get %s: %s", id, strings.TrimSpace(stderr.String()))
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
