package session

import (
	"sync"
	"time"

	"github.com/sters/cc-session-tailing/internal/parser"
)

// Session represents a single session's state.
type Session struct {
	ID         string
	Path       string
	IsSubagent bool
	Messages   []parser.Message
	Offset     int64
	LastUpdate time.Time
}

// Manager manages sessions and panel assignments using LRU.
type Manager struct {
	mu          sync.RWMutex
	panels      int
	sessions    map[string]*Session
	panelAssign map[int]string // panelIndex -> sessionID
}

// NewManager creates a new session manager.
func NewManager(panels int) *Manager {
	return &Manager{
		panels:      panels,
		sessions:    make(map[string]*Session),
		panelAssign: make(map[int]string),
	}
}

// GetOrCreateSession gets or creates a session.
func (m *Manager) GetOrCreateSession(sessionID, path string, isSubagent bool) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[sessionID]; ok {
		s.LastUpdate = time.Now()
		return s
	}

	s := &Session{
		ID:         sessionID,
		Path:       path,
		IsSubagent: isSubagent,
		Messages:   nil,
		Offset:     0,
		LastUpdate: time.Now(),
	}
	m.sessions[sessionID] = s

	// Assign to a panel
	m.assignPanel(sessionID)

	return s
}

// UpdateSession updates a session with new messages.
func (m *Manager) UpdateSession(sessionID string, messages []parser.Message, newOffset int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	s, ok := m.sessions[sessionID]
	if !ok {
		return
	}

	s.Messages = append(s.Messages, messages...)
	s.Offset = newOffset
	s.LastUpdate = time.Now()
}

// assignPanel assigns a panel to a session using LRU.
func (m *Manager) assignPanel(sessionID string) {
	// Check if already assigned
	for _, sid := range m.panelAssign {
		if sid == sessionID {
			return
		}
	}

	// Find an empty panel
	for i := 0; i < m.panels; i++ {
		if _, ok := m.panelAssign[i]; !ok {
			m.panelAssign[i] = sessionID
			return
		}
	}

	// All panels are full, find the oldest session
	oldestPanel := m.getOldestPanel()
	if oldestPanel >= 0 {
		m.panelAssign[oldestPanel] = sessionID
	}
}

// getOldestPanel returns the panel with the oldest session.
func (m *Manager) getOldestPanel() int {
	var oldestPanel = -1
	var oldestTime time.Time

	for panel, sessionID := range m.panelAssign {
		s, ok := m.sessions[sessionID]
		if !ok {
			return panel // Empty session, use this panel
		}

		if oldestPanel == -1 || s.LastUpdate.Before(oldestTime) {
			oldestPanel = panel
			oldestTime = s.LastUpdate
		}
	}

	return oldestPanel
}

// GetPanelSessions returns sessions for each panel.
func (m *Manager) GetPanelSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, m.panels)
	for i := 0; i < m.panels; i++ {
		if sessionID, ok := m.panelAssign[i]; ok {
			if s, ok := m.sessions[sessionID]; ok {
				result[i] = s
			}
		}
	}
	return result
}

// GetSession returns a session by ID.
func (m *Manager) GetSession(sessionID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[sessionID]
}

// PanelCount returns the number of panels.
func (m *Manager) PanelCount() int {
	return m.panels
}
