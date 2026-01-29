package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/sters/cc-session-tailing/internal/parser"
	"github.com/sters/cc-session-tailing/internal/session"
	"github.com/sters/cc-session-tailing/internal/tui"
	"github.com/sters/cc-session-tailing/internal/watcher"
)

//nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	panels      int
	projectPath string
)

var rootCmd = &cobra.Command{
	Use:   "cc-session-tailing",
	Short: "Real-time TUI viewer for Claude Code session logs",
	Long: `A TUI application that monitors Claude Code session logs in real-time.
Shows session activity in multiple panels with LRU-based panel assignment.`,
	RunE: runTUI,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version:    %s\n", getVersion())
		fmt.Printf("Commit:     %s\n", getCommit())
		fmt.Printf("Built:      %s\n", getDate())
		fmt.Printf("Go version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.Flags().IntVarP(&panels, "panels", "p", 4, "Number of panels to display")
	rootCmd.Flags().StringVarP(&projectPath, "project", "d", ".", "Project directory to watch")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Resolve project path
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	// Build the Claude projects path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Convert absolute path to Claude's project path format
	// ~/.claude/projects/{project-path}/
	claudeProjectPath := filepath.Join(homeDir, ".claude", "projects", pathToClaudePath(absProjectPath))

	// Check if the directory exists
	if _, err := os.Stat(claudeProjectPath); os.IsNotExist(err) {
		return fmt.Errorf("claude project directory does not exist: %s\nMake sure Claude Code has been used in this project", claudeProjectPath)
	}

	// Create watcher
	w, err := watcher.New(claudeProjectPath)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start watcher
	if err := w.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer w.Stop()

	// Create session manager
	manager := session.NewManager(panels)

	// Scan existing files
	existingEvents, err := w.ScanExisting()
	if err != nil {
		return fmt.Errorf("failed to scan existing files: %w", err)
	}

	// Process existing files
	for _, event := range existingEvents {
		sess := manager.GetOrCreateSession(event.SessionID, event.Path, event.IsSubagent)
		messages, newOffset, err := parser.ParseFromOffset(event.Path, sess.Offset)
		if err == nil && len(messages) > 0 {
			manager.UpdateSession(event.SessionID, messages, newOffset)
		}
	}

	// Create TUI model
	model := tui.NewModel(manager, w)

	// Run bubbletea program
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// pathToClaudePath converts an absolute path to Claude's path format.
// e.g., /Users/foo/github.com/project -> -Users-foo-github-com-project
func pathToClaudePath(absPath string) string {
	// Replace path separators with dashes
	// The leading dash is kept (e.g., /Users/foo -> -Users-foo)
	result := strings.ReplaceAll(absPath, string(filepath.Separator), "-")
	// Also replace dots with dashes (e.g., github.com -> github-com)
	result = strings.ReplaceAll(result, ".", "-")
	return result
}

func getVersion() string {
	if version != "dev" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	return version
}

func getCommit() string {
	if commit != "none" {
		return commit
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				if len(setting.Value) > 7 {
					return setting.Value[:7]
				}

				return setting.Value
			}
		}
	}

	return commit
}

func getDate() string {
	if date != "unknown" {
		return date
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.time" {
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					return t.UTC().Format("2006-01-02T15:04:05Z")
				}

				return setting.Value
			}
		}
	}

	return date
}
