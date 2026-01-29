package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)
)

// View renders the TUI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// Calculate panel dimensions
	panels := m.manager.PanelCount()
	panelWidth := m.width / panels
	panelHeight := m.height - 2 // Leave room for help line

	// Get sessions for each panel
	sessions := m.manager.GetPanelSessions()

	// Render each panel
	var panelViews []string
	for i := 0; i < panels; i++ {
		scrollPos := 0
		if i < len(m.scrollPos) {
			scrollPos = m.scrollPos[i]
		}

		var sess = sessions[i]
		panel := RenderPanel(sess, panelWidth, panelHeight, scrollPos)
		panelViews = append(panelViews, panel)
	}

	// Join panels horizontally
	panelsRow := lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)

	// Help line
	help := helpStyle.Render("q: quit | j/k: scroll | Watching for sessions...")

	return lipgloss.JoinVertical(lipgloss.Left, panelsRow, help)
}

// RenderWelcome renders a welcome message when no sessions are active.
func RenderWelcome(width, height int) string {
	msg := []string{
		"Claude Code Session Tailing",
		"",
		"Waiting for session activity...",
		"",
		"Start Claude Code in another terminal to see logs here.",
		"",
		"Press 'q' to quit",
	}

	content := strings.Join(msg, "\n")
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
