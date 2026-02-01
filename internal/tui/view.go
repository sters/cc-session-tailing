package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the TUI.
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.viewMode == ViewModeTree {
		return m.renderTreeView()
	}

	return m.renderPanelView()
}

func (m *Model) renderTreeView() string {
	return m.treeView.View()
}

func (m *Model) renderPanelView() string {
	// Calculate panel dimensions.
	panels := m.manager.PanelCount()
	panelWidth := m.width / panels
	panelHeight := m.height - 2 // Leave room for help line.

	// Get sessions for each panel.
	sessions := m.manager.GetPanelSessions()

	// Render each panel.
	panelViews := make([]string, 0, panels)
	for i := range panels {
		scrollPos := 0
		if i < len(m.scrollPos) {
			scrollPos = m.scrollPos[i]
		}

		sess := sessions[i]
		panel := m.renderer.RenderPanel(sess, panelWidth, panelHeight, scrollPos)
		panelViews = append(panelViews, panel)
	}

	// Join panels horizontally.
	panelsRow := lipgloss.JoinHorizontal(lipgloss.Top, panelViews...)

	// Help line.
	help := m.renderer.styles.HelpStyle.Render("q: quit | j/k: scroll | p: panels (%d) | t: tree mode | Watching for sessions...")
	help = fmt.Sprintf(help, panels)

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
