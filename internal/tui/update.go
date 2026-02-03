package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "t":
			cmd := m.ToggleViewMode()

			return m, cmd
		}

		// Mode-specific key handling.
		if m.viewMode == ViewModeTree {
			return m.updateTreeMode(msg)
		}

		return m.updatePanelMode(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		wasReady := m.ready
		m.ready = true
		m.treeView.SetSize(m.width, m.height)

		// Initialize tree view on first ready.
		if !wasReady && m.viewMode == ViewModeTree {
			cmd := m.treeView.RefreshSessions()

			return m, cmd
		}

	case FileUpdateMsg:
		m.processFileUpdate(msg.Event)
		// Refresh tree view if in tree mode.
		if m.viewMode == ViewModeTree {
			highlightCmd := m.treeView.RefreshSessions()
			m.treeView.RefreshLog()

			return m, tea.Batch(waitForFileEvents(m.watcher), highlightCmd)
		}

		return m, waitForFileEvents(m.watcher)

	case HighlightClearMsg:
		// Clear highlights in tree view.
		if m.viewMode == ViewModeTree {
			m.treeView.ClearHighlights()
		}

		return m, nil
	}

	return m, nil
}

func (m *Model) updateTreeMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd := m.treeView.Update(msg)

	return m, cmd
}

func (m *Model) updatePanelMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.scrollDown()
	case "k", "up":
		m.scrollUp()
	case "p":
		m.cyclePanelCount()
	}

	return m, nil
}

func (m *Model) cyclePanelCount() {
	current := m.manager.PanelCount()
	next := current + 1
	if next > 5 {
		next = 1
	}
	m.manager.SetPanelCount(next)
	// Resize scrollPos array with -1 (follow bottom mode).
	m.scrollPos = make([]int, next)
	for i := range m.scrollPos {
		m.scrollPos[i] = -1
	}
}

func (m *Model) scrollDown() {
	// Scroll down = show newer content.
	// If in fixed mode (scrollPos >= 0), move start line forward.
	// If we reach the bottom, switch to follow mode (scrollPos = -1).
	sessions := m.manager.GetPanelSessions()
	panels := m.manager.PanelCount()
	panelHeight := m.height - 2 - 2 - 1 // total height - help line - border - header

	for i := range m.scrollPos {
		if i >= panels {
			continue
		}
		if m.scrollPos[i] < 0 {
			// Already in follow mode.
			continue
		}
		sess := sessions[i]
		if sess == nil {
			continue
		}
		// Estimate total lines.
		totalLines := len(sess.Messages) * 3
		maxStartLine := totalLines - panelHeight
		if maxStartLine < 0 {
			maxStartLine = 0
		}

		m.scrollPos[i]++
		// If we've scrolled past the bottom, switch to follow mode.
		if m.scrollPos[i] >= maxStartLine {
			m.scrollPos[i] = -1
		}
	}
}

func (m *Model) scrollUp() {
	// Scroll up = show older content.
	// If in follow mode (scrollPos = -1), calculate current start line and fix it.
	// Then move start line backward.
	sessions := m.manager.GetPanelSessions()
	panels := m.manager.PanelCount()
	panelHeight := m.height - 2 - 2 - 1 // total height - help line - border - header

	for i := range m.scrollPos {
		if i >= panels {
			continue
		}
		sess := sessions[i]
		if sess == nil {
			continue
		}
		// Estimate total lines.
		totalLines := len(sess.Messages) * 3
		maxStartLine := totalLines - panelHeight
		if maxStartLine < 0 {
			maxStartLine = 0
		}

		if m.scrollPos[i] < 0 {
			// Currently in follow mode, switch to fixed mode at current position.
			m.scrollPos[i] = maxStartLine
		}

		// Move start line backward (show older content).
		if m.scrollPos[i] > 0 {
			m.scrollPos[i]--
		}
	}
}
