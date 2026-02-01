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
	ParentID   string // Parent session ID (empty for root sessions)
	IsSubagent bool
	Messages   []parser.Message
	Offset     int64
	LastUpdate time.Time
}

// Node represents a session with its children for tree display.
type Node struct {
	Session  *Session
	Children []*Node
	Expanded bool
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

// GetOrCreateSessionWithParent gets or creates a session with a parent.
func (m *Manager) GetOrCreateSessionWithParent(sessionID, path, parentID string, isSubagent bool) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[sessionID]; ok {
		s.LastUpdate = time.Now()

		return s
	}

	s := &Session{
		ID:         sessionID,
		Path:       path,
		ParentID:   parentID,
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

	// Find an empty panel.
	for i := range m.panels {
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
	oldestPanel := -1
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
	for i := range m.panels {
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

// GetAllSessions returns all sessions sorted by last update time (newest first).
func (m *Manager) GetAllSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}

	// Sort by LastUpdate descending (newest first).
	for i := range len(result) - 1 {
		for j := i + 1; j < len(result); j++ {
			if result[j].LastUpdate.After(result[i].LastUpdate) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// GetSessionTree returns sessions as a tree structure.
func (m *Manager) GetSessionTree() []*Node {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Build a map of parent -> children.
	childrenMap := make(map[string][]*Session)
	var roots []*Session

	for _, s := range m.sessions {
		if s.ParentID == "" {
			roots = append(roots, s)
		} else {
			childrenMap[s.ParentID] = append(childrenMap[s.ParentID], s)
		}
	}

	// Sort roots by LastUpdate descending.
	for i := range len(roots) - 1 {
		for j := i + 1; j < len(roots); j++ {
			if roots[j].LastUpdate.After(roots[i].LastUpdate) {
				roots[i], roots[j] = roots[j], roots[i]
			}
		}
	}

	// Build tree nodes.
	result := make([]*Node, 0, len(roots))
	for _, root := range roots {
		node := m.buildNode(root, childrenMap)
		result = append(result, node)
	}

	return result
}

func (m *Manager) buildNode(s *Session, childrenMap map[string][]*Session) *Node {
	node := &Node{
		Session:  s,
		Expanded: true,
	}

	children := childrenMap[s.ID]
	// Sort children by LastUpdate descending.
	for i := range len(children) - 1 {
		for j := i + 1; j < len(children); j++ {
			if children[j].LastUpdate.After(children[i].LastUpdate) {
				children[i], children[j] = children[j], children[i]
			}
		}
	}

	for _, child := range children {
		childNode := m.buildNode(child, childrenMap)
		node.Children = append(node.Children, childNode)
	}

	return node
}

// GetChildSessions returns child sessions of a given session.
func (m *Manager) GetChildSessions(parentID string) []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var children []*Session
	for _, s := range m.sessions {
		if s.ParentID == parentID {
			children = append(children, s)
		}
	}

	// Sort by LastUpdate descending.
	for i := range len(children) - 1 {
		for j := i + 1; j < len(children); j++ {
			if children[j].LastUpdate.After(children[i].LastUpdate) {
				children[i], children[j] = children[j], children[i]
			}
		}
	}

	return children
}
