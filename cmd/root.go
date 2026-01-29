package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/sters/cc-session-tailing/internal/parser"
	"github.com/sters/cc-session-tailing/internal/session"
	"github.com/sters/cc-session-tailing/internal/tui"
	"github.com/sters/cc-session-tailing/internal/watcher"
)

// ProjectNotFoundError is returned when the Claude project directory does not exist.
type ProjectNotFoundError struct {
	Path string
}

func (e *ProjectNotFoundError) Error() string {
	return fmt.Sprintf("claude project directory does not exist: %s\nMake sure Claude Code has been used in this project", e.Path)
}

// CLI holds the command line interface state.
type CLI struct {
	panels      int
	projectPath string
	rootCmd     *cobra.Command
}

// NewCLI creates a new CLI instance.
func NewCLI() *CLI {
	cli := &CLI{}
	cli.rootCmd = &cobra.Command{
		Use:   "cc-session-tailing",
		Short: "Real-time TUI viewer for Claude Code session logs",
		Long: `A TUI application that monitors Claude Code session logs in real-time.
Shows session activity in multiple panels with LRU-based panel assignment.`,
		RunE: cli.runTUI,
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("Version:    %s\n", getVersion())
			fmt.Printf("Commit:     %s\n", getCommit())
			fmt.Printf("Built:      %s\n", getDate())
			fmt.Printf("Go version: %s\n", runtime.Version())
			fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
		},
	}

	cli.rootCmd.AddCommand(versionCmd)
	cli.rootCmd.Flags().IntVarP(&cli.panels, "panels", "p", 4, "Number of panels to display")
	cli.rootCmd.Flags().StringVarP(&cli.projectPath, "project", "d", ".", "Project directory to watch")

	return cli
}

// Execute runs the root command.
func Execute() {
	cli := NewCLI()
	if err := cli.rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func (cli *CLI) runTUI(_ *cobra.Command, _ []string) error {
	// Resolve project path.
	absProjectPath, err := filepath.Abs(cli.projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	// Build the Claude projects path.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Convert absolute path to Claude's project path format.
	// ~/.claude/projects/{project-path}/
	claudeProjectPath := filepath.Join(homeDir, ".claude", "projects", pathToClaudePath(absProjectPath))

	// Check if the directory exists.
	if _, err := os.Stat(claudeProjectPath); os.IsNotExist(err) {
		return &ProjectNotFoundError{Path: claudeProjectPath}
	}

	// Create watcher.
	w, err := watcher.New(claudeProjectPath)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start watcher.
	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer func() { _ = w.Stop() }()

	// Create session manager.
	manager := session.NewManager(cli.panels)

	// Scan existing files.
	existingEvents, err := w.ScanExisting()
	if err != nil {
		return fmt.Errorf("failed to scan existing files: %w", err)
	}

	// Process existing files.
	for _, event := range existingEvents {
		sess := manager.GetOrCreateSession(event.SessionID, event.Path, event.IsSubagent)
		messages, newOffset, err := parser.ParseFromOffset(event.Path, sess.Offset)
		if err == nil && len(messages) > 0 {
			manager.UpdateSession(event.SessionID, messages, newOffset)
		}
	}

	// Create TUI model.
	model := tui.NewModel(manager, w)

	// Run bubbletea program.
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// pathToClaudePath converts an absolute path to Claude's path format.
// e.g., /Users/foo/github.com/project -> -Users-foo-github-com-project.
func pathToClaudePath(absPath string) string {
	// Replace path separators with dashes.
	// The leading dash is kept (e.g., /Users/foo -> -Users-foo).
	result := strings.ReplaceAll(absPath, string(filepath.Separator), "-")
	// Also replace dots with dashes (e.g., github.com -> github-com).
	result = strings.ReplaceAll(result, ".", "-")

	return result
}
