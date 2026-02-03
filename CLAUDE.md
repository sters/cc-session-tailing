# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cc-session-tailing is a real-time TUI viewer for monitoring Claude Code session logs. It watches `~/.claude/projects/{project-path}/` for JSONL session files and displays them with a multi-panel interface supporting 1-5 simultaneous sessions.

## Build Commands

```bash
make build     # Build binary to bin/app
make run       # Run with version ldflags
make test      # Run tests with -race flag
make lint      # Run golangci-lint (strict mode)
make lint-fix  # Auto-fix lint issues
make cover     # Generate coverage report
make tidy      # go mod tidy
```

## Architecture

### Package Structure

- **`cmd/`** - CLI entry point using Cobra. Handles flags (`--panels`, `--project`, `--mode`), initializes components, and converts project paths to Claude's format (`/path/to/project` → `-path-to-project`).

- **`internal/parser/`** - JSONL file parsing with offset-based incremental reading. `ParseFromOffset()` enables efficient real-time updates by only parsing new content.

- **`internal/session/`** - Thread-safe session management (RWMutex). Implements LRU panel assignment - new sessions evict the least-recently-updated session when all panels are full. Supports parent-child relationships for subagent sessions.

- **`internal/watcher/`** - File system monitoring using fsnotify. Detects root sessions (`{session-id}.jsonl`) and subagent sessions (`{session-id}/subagents/agent-{id}.jsonl`).

- **`internal/tui/`** - Bubble Tea TUI framework implementation:
  - `model.go` - Main model holding state
  - `update.go` - Key handlers and message processing
  - `view.go` - Renders panel or tree view
  - `panel.go` - Panel rendering with color-coded message types
  - `components/` - Reusable viewport and tree components

### Key Patterns

1. **Model-View-Update (MVU)**: Bubble Tea's unidirectional data flow
2. **LRU Cache**: Panel assignment evicts oldest session when full
3. **Offset-Based Parsing**: Only reads new content from files
4. **Event-Driven**: Watcher sends events via channels to TUI

### Message Type Colors

- Blue: `[USER]` - User input
- Gray/italic: `[THINK]` - Thinking process
- White: `[TEXT]` - Text responses
- Orange/bold: `[TOOL]` - Tool invocations

## Development Process

1. **Before committing**: Run `make lint` and `make test` locally
2. **CI runs on every push**: Lint → Test with coverage → Upload coverage report
3. **Release**: Tag with `v*.*.*` format triggers GoReleaser to build cross-platform binaries

## Key Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/fsnotify/fsnotify` - File watching
- `github.com/spf13/cobra` - CLI framework
