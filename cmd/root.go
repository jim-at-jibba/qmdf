package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jim-at-jibba/qmdf/internal/config"
	"github.com/jim-at-jibba/qmdf/internal/qmd"
	"github.com/jim-at-jibba/qmdf/internal/tui"
)

var (
	flagCollection string
	flagMode       string
	flagResults    int
	flagMinScore   float64
	flagNoPreview  bool
	flagPrint      bool
	flagNoQmdCheck bool

	appVersion string // set by Execute()
)

var rootCmd = &cobra.Command{
	Use:   "qmdf",
	Short: "Interactive TUI wrapper for qmd — a Markdown document search tool",
	Long: `qmdf is a fast, zero-dependency TUI that wraps the qmd CLI.

Type to search your Markdown document collection. Navigate results with
↑/↓ (or j/k), press Enter to open in $EDITOR, Tab to cycle search modes.

Search modes:
  search   — full-text keyword search
  vsearch  — vector/semantic search
  query    — LLM-reranked query (slower, most relevant)

Install qmd first:  npm install -g @tobilu/qmd`,
	RunE: runTUI,
}

// Execute runs the root command.
func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&flagCollection, "collection", "c", "", "qmd collection name")
	rootCmd.Flags().StringVarP(&flagMode, "mode", "m", "", "search mode: search|vsearch|query")
	rootCmd.Flags().IntVarP(&flagResults, "results", "n", 0, "max results (0 = use config default)")
	rootCmd.Flags().Float64Var(&flagMinScore, "min-score", 0, "minimum relevance score (0.0–1.0)")
	rootCmd.Flags().BoolVar(&flagNoPreview, "no-preview", false, "disable preview pane")
	rootCmd.Flags().BoolVar(&flagPrint, "print", false, "print selected file path to stdout (shell integration)")
	rootCmd.Flags().BoolVar(&flagNoQmdCheck, "no-qmd-check", false, "skip qmd installation check")
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Load config file
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI flag overrides (flags beat config file)
	if cmd.Flags().Changed("collection") {
		cfg.Collection = flagCollection
	}
	if cmd.Flags().Changed("mode") {
		cfg.Mode = flagMode
	}
	if cmd.Flags().Changed("results") {
		cfg.Results = flagResults
	}
	if cmd.Flags().Changed("min-score") {
		cfg.MinScore = flagMinScore
	}
	if cmd.Flags().Changed("no-preview") {
		cfg.NoPreview = flagNoPreview
	}
	cfg.PrintMode = flagPrint

	// Validate mode
	if cfg.Mode != "" {
		switch qmd.Mode(cfg.Mode) {
		case qmd.ModeSearch, qmd.ModeVSearch, qmd.ModeQuery:
			// OK
		default:
			return fmt.Errorf("invalid mode %q — must be search, vsearch, or query", cfg.Mode)
		}
	}

	// Check qmd is installed (unless suppressed)
	if !flagNoQmdCheck {
		if err := qmd.CheckInstalled(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
			fmt.Fprintln(os.Stderr, "Results will be empty until qmd is installed.")
		}
	}

	// Detect dark/light background NOW — before tea.NewProgram takes over
	// stdin/stdout. AdaptiveColor must never be used inside the TUI event loop
	// because the OSC terminal query response would be read as keyboard input.
	lipsRenderer := lipgloss.NewRenderer(os.Stdout)
	isDark := lipsRenderer.HasDarkBackground()
	tui.InitStyles()

	// Build and run the TUI
	model := tui.New(cfg, isDark, appVersion)

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if flagPrint {
		// In print mode we want mouse disabled and plain output — altscreen is
		// still fine because we restore the terminal on exit.
		opts = append(opts, tea.WithMouseCellMotion())
	}

	p := tea.NewProgram(model, opts...)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}
