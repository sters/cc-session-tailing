package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.scrollDown()
		case "k", "up":
			m.scrollUp()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case FileUpdateMsg:
		m.processFileUpdate(msg.Event)
		return m, waitForFileEvents(m.watcher)
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
