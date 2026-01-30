package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Event represents a file event.
type Event struct {
	Path       string
	SessionID  string
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
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
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
	// Add the project directory.
	if err := w.addDirRecursive(w.projectPath); err != nil {
		return err
	}

	go w.run()

	return nil
}

// Stop stops the watcher.
func (w *Watcher) Stop() error {
	close(w.done)

	if err := w.fsWatcher.Close(); err != nil {
		return fmt.Errorf("failed to close fsnotify watcher: %w", err)
	}

	return nil
}

func (w *Watcher) addDirRecursive(dir string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil {
			// Skip inaccessible paths by returning filepath.SkipDir for dirs.
			return filepath.SkipDir
		}
		if info.IsDir() {
			// Ignore errors when adding watch - directory may be removed.
			_ = w.fsWatcher.Add(path)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	return nil
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

	// Handle new directories.
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

	// Main session: {session-id}.jsonl.
	if len(parts) == 1 {
		sessionID := strings.TrimSuffix(parts[0], ".jsonl")

		return sessionID, false
	}

	// Subagent: {session-id}/subagents/agent-{id}.jsonl.
	if len(parts) >= 3 && parts[1] == "subagents" {
		agentFile := strings.TrimSuffix(parts[len(parts)-1], ".jsonl")
		sessionID := parts[0] + "/" + agentFile

		return sessionID, true
	}

	return "", false
}

// eventWithModTime holds an event with its file modification time for sorting.
type eventWithModTime struct {
	event   Event
	modTime int64
}

// ScanExisting scans for existing JSONL files and returns them sorted by modification time (oldest first).
func (w *Watcher) ScanExisting() ([]Event, error) {
	var eventsWithTime []eventWithModTime

	err := filepath.Walk(w.projectPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil {
			// Skip inaccessible paths.
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}

		sessionID, isSubagent := w.parseSessionInfo(path)
		if sessionID != "" {
			eventsWithTime = append(eventsWithTime, eventWithModTime{
				event: Event{
					Path:       path,
					SessionID:  sessionID,
					IsSubagent: isSubagent,
				},
				modTime: info.ModTime().UnixNano(),
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan existing files: %w", err)
	}

	// Sort by modification time (oldest first) so that newest sessions end up in panels.
	sort.Slice(eventsWithTime, func(i, j int) bool {
		return eventsWithTime[i].modTime < eventsWithTime[j].modTime
	})

	// Extract events in sorted order.
	events := make([]Event, len(eventsWithTime))
	for i, e := range eventsWithTime {
		events[i] = e.event
	}

	return events, nil
}
