package watcher

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Event represents a file event.
type Event struct {
	Path      string
	SessionID string
	IsSubagent bool
}

// Watcher monitors a project directory for JSONL file changes.
type Watcher struct {
	fsWatcher   *fsnotify.Watcher
	projectPath string
	Events      chan Event
	Errors      chan error
	done        chan struct{}
}

// New creates a new watcher for the given project path.
func New(projectPath string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		fsWatcher:   fsWatcher,
		projectPath: projectPath,
		Events:      make(chan Event, 100),
		Errors:      make(chan error, 10),
		done:        make(chan struct{}),
	}

	return w, nil
}

// Start begins watching the project directory.
func (w *Watcher) Start() error {
	// Add the project directory
	if err := w.addDirRecursive(w.projectPath); err != nil {
		return err
	}

	go w.run()
	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	close(w.done)
	return w.fsWatcher.Close()
}

func (w *Watcher) addDirRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			if err := w.fsWatcher.Add(path); err != nil {
				return nil // Skip errors
			}
		}
		return nil
	})
}

func (w *Watcher) run() {
	for {
		select {
		case <-w.done:
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			select {
			case w.Errors <- err:
			default:
			}
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	// Handle new directories
	if event.Op&fsnotify.Create != 0 {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			_ = w.addDirRecursive(path)
			return
		}
	}

	// Only process .jsonl files
	if !strings.HasSuffix(path, ".jsonl") {
		return
	}

	// Only process write and create events
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return
	}

	// Parse session ID and check if subagent
	sessionID, isSubagent := w.parseSessionInfo(path)
	if sessionID == "" {
		return
	}

	select {
	case w.Events <- Event{
		Path:       path,
		SessionID:  sessionID,
		IsSubagent: isSubagent,
	}:
	default:
	}
}

func (w *Watcher) parseSessionInfo(path string) (string, bool) {
	rel, err := filepath.Rel(w.projectPath, path)
	if err != nil {
		return "", false
	}

	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 {
		return "", false
	}

	// Main session: {session-id}.jsonl
	if len(parts) == 1 {
		sessionID := strings.TrimSuffix(parts[0], ".jsonl")
		return sessionID, false
	}

	// Subagent: {session-id}/subagents/agent-{id}.jsonl
	if len(parts) >= 3 && parts[1] == "subagents" {
		agentFile := strings.TrimSuffix(parts[len(parts)-1], ".jsonl")
		sessionID := parts[0] + "/" + agentFile
		return sessionID, true
	}

	return "", false
}

// ScanExisting scans for existing JSONL files.
func (w *Watcher) ScanExisting() ([]Event, error) {
	var events []Event

	err := filepath.Walk(w.projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		sessionID, isSubagent := w.parseSessionInfo(path)
		if sessionID != "" {
			events = append(events, Event{
				Path:       path,
				SessionID:  sessionID,
				IsSubagent: isSubagent,
			})
		}
		return nil
	})

	return events, err
}
