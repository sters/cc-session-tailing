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
			m.ToggleViewMode()

			return m, nil
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
			m.treeView.RefreshSessions()
		}

	case FileUpdateMsg:
		m.processFileUpdate(msg.Event)
		// Refresh tree view if in tree mode.
		if m.viewMode == ViewModeTree {
			m.treeView.RefreshSessions()
			m.treeView.RefreshLog()
		}

		return m, waitForFileEvents(m.watcher)
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
	}

	return m, nil
}

func (m *Model) scrollDown() {
	// Scroll down = show newer content = decrease scrollPos
	for i := range m.scrollPos {
		if m.scrollPos[i] > 0 {
			m.scrollPos[i]--
		}
	}
}

func (m *Model) scrollUp() {
	// Scroll up = show older content = increase scrollPos
	for i := range m.scrollPos {
		m.scrollPos[i]++
	}
}
