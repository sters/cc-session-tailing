package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sters/cc-session-tailing/internal/parser"
	"github.com/sters/cc-session-tailing/internal/session"
	"github.com/sters/cc-session-tailing/internal/watcher"
)

// FileUpdateMsg is sent when a file is updated.
type FileUpdateMsg struct {
	Event watcher.Event
}

// Model is the bubbletea model for the TUI.
type Model struct {
	manager   *session.Manager
	watcher   *watcher.Watcher
	renderer  *Renderer
	width     int
	height    int
	scrollPos []int // scroll position for each panel
	ready     bool
}

// NewModel creates a new TUI model.
func NewModel(manager *session.Manager, w *watcher.Watcher) *Model {
	panels := manager.PanelCount()

	return &Model{
		manager:   manager,
		watcher:   w,
		renderer:  NewRenderer(NewStyles()),
		scrollPos: make([]int, panels),
	}
}

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		waitForFileEvents(m.watcher),
	)
}

// waitForFileEvents waits for file events from the watcher.
func waitForFileEvents(w *watcher.Watcher) tea.Cmd {
	return func() tea.Msg {
		select {
		case event := <-w.Events:
			return FileUpdateMsg{Event: event}
		case <-w.Errors:
			return nil
		}
	}
}

// processFileUpdate processes a file update event.
func (m *Model) processFileUpdate(event watcher.Event) {
	// Get or create session
	sess := m.manager.GetOrCreateSession(event.SessionID, event.Path, event.IsSubagent)

	// Parse new messages from the file
	messages, newOffset, err := parser.ParseFromOffset(event.Path, sess.Offset)
	if err != nil {
		return
	}

	if len(messages) > 0 {
		m.manager.UpdateSession(event.SessionID, messages, newOffset)
	}
}
