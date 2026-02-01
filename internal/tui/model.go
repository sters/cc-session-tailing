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
	viewMode  ViewMode
	treeView  *TreeView
}

// NewModel creates a new TUI model with panel mode.
func NewModel(manager *session.Manager, w *watcher.Watcher) *Model {
	panels := manager.PanelCount()

	return &Model{
		manager:   manager,
		watcher:   w,
		renderer:  NewRenderer(NewStyles()),
		scrollPos: make([]int, panels),
		viewMode:  ViewModePanel,
		treeView:  NewTreeView(manager),
	}
}

// NewModelWithMode creates a new TUI model with the specified view mode.
func NewModelWithMode(manager *session.Manager, w *watcher.Watcher, mode ViewMode) *Model {
	panels := manager.PanelCount()

	return &Model{
		manager:   manager,
		watcher:   w,
		renderer:  NewRenderer(NewStyles()),
		scrollPos: make([]int, panels),
		viewMode:  mode,
		treeView:  NewTreeView(manager),
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
	var sess *session.Session
	if event.ParentID != "" {
		sess = m.manager.GetOrCreateSessionWithParent(event.SessionID, event.Path, event.ParentID, event.IsSubagent)
	} else {
		sess = m.manager.GetOrCreateSession(event.SessionID, event.Path, event.IsSubagent)
	}

	// Parse new messages from the file
	messages, newOffset, err := parser.ParseFromOffset(event.Path, sess.Offset)
	if err != nil {
		return
	}

	if len(messages) > 0 {
		m.manager.UpdateSession(event.SessionID, messages, newOffset)
	}
}

// ViewMode returns the current view mode.
func (m *Model) ViewMode() ViewMode {
	return m.viewMode
}

// SetViewMode sets the view mode.
func (m *Model) SetViewMode(mode ViewMode) {
	m.viewMode = mode
}

// ToggleViewMode toggles between tree and panel modes.
func (m *Model) ToggleViewMode() {
	if m.viewMode == ViewModeTree {
		m.viewMode = ViewModePanel
	} else {
		m.viewMode = ViewModeTree
		m.treeView.RefreshSessions()
	}
}
