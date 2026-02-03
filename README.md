# cc-session-tailing

[![go](https://github.com/sters/cc-session-tailing/workflows/Go/badge.svg)](https://github.com/sters/cc-session-tailing/actions?query=workflow%3AGo)
[![coverage](docs/coverage.svg)](https://github.com/sters/cc-session-tailing)
[![go-report](https://goreportcard.com/badge/github.com/sters/cc-session-tailing)](https://goreportcard.com/report/github.com/sters/cc-session-tailing)

A real-time TUI (Terminal User Interface) viewer for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) session logs. Monitor multiple Claude Code sessions simultaneously with an intuitive multi-panel display.

## Features

- **Real-time Monitoring**: Watch Claude Code session logs as they happen
- **Multi-panel Display**: View multiple sessions side-by-side (1-5 panels, dynamically adjustable)
- **LRU Panel Assignment**: Most recently updated session always appears in the leftmost panel
- **Tree View Mode**: Hierarchical view showing parent-child session relationships
- **Message Type Highlighting**: Different colors for thinking, text, tool usage, and user messages
- **Subagent Support**: Displays both main sessions and subagent sessions with hierarchy
- **Scrollbar**: Visual indicator for scroll position within each panel
- **Keyboard Navigation**: Scroll through session history with vim-style keybindings

## Installation

### Using Go

```shell
go install github.com/sters/cc-session-tailing@latest
```

### From Releases

Download the appropriate binary for your platform from [Releases](https://github.com/sters/cc-session-tailing/releases).

## Usage

Navigate to your project directory and run:

```shell
cc-session-tailing
```

The tool automatically finds Claude Code session logs in `~/.claude/projects/<project-path>/`.

### Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--mode` | `-m` | `tree` | View mode: `tree` or `panel` (defaults to `panel` if `-p` is specified) |
| `--panels` | `-p` | `4` | Number of panels to display (panel mode) |
| `--project` | `-d` | `.` | Project directory to watch |

### Examples

```shell
# Watch the current directory in tree mode (default)
cc-session-tailing

# Watch in panel mode with 2 panels
cc-session-tailing -p 2

# Explicitly set tree mode
cc-session-tailing -m tree

# Watch a specific project directory
cc-session-tailing -d /path/to/your/project

# Combine options
cc-session-tailing -p 5 -d ~/projects/my-app
```

### Keyboard Shortcuts

#### Common

| Key | Action |
|-----|--------|
| `q` / `Ctrl+C` | Quit |
| `t` | Toggle between tree mode and panel mode |

#### Tree Mode

| Key | Action |
|-----|--------|
| `j` / `Down` | Move selection down (tree) / Scroll down (log) |
| `k` / `Up` | Move selection up (tree) / Scroll up (log) |
| `Enter` | Switch focus to log viewport |
| `Esc` | Return focus to session tree |
| `f` | Toggle fullscreen log (when log is focused) |

#### Panel Mode

| Key | Action |
|-----|--------|
| `j` / `Down` | Scroll down (show newer messages) |
| `k` / `Up` | Scroll up (show older messages) |
| `p` | Cycle panel count (1 → 2 → 3 → 4 → 5 → 1) |

## How It Works

1. The tool monitors the Claude Code session directory (`~/.claude/projects/<project-path>/`)
2. When Claude Code is active, it writes session logs as JSONL files
3. This tool watches for file changes and parses new messages in real-time
4. Sessions are displayed in panels, sorted by last update time (newest on the left)
5. When panel count increases, unassigned sessions are automatically loaded into new panels

### Message Types

- **[USER]**: User input messages (blue)
- **[THINK]**: Claude's thinking process (gray, italic)
- **[TEXT]**: Claude's text responses (white)
- **[TOOL]**: Tool invocations with parameters (orange)
- **[RESULT]**: Tool execution results

## Requirements

- Go 1.24 or later (for building from source)
- Claude Code must have been used in the target project at least once

## License

MIT License - see [LICENSE](LICENSE) for details.
