package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sters/cc-session-tailing/internal/session"
	"github.com/sters/cc-session-tailing/internal/tui/components"
)

// Focus represents which component has focus in tree view.
type Focus int

const (
	// FocusTree means the session tree has focus.
	FocusTree Focus = iota
	// FocusLog means the log viewport has focus.
	FocusLog
)

// TreeView manages the tree view layout with session tree and log viewport.
type TreeView struct {
	tree       *components.SessionTree
	log        *components.LogViewport
	focus      Focus
	treeHidden bool
	width      int
	height     int
	manager    *session.Manager
	renderer   *Renderer
}

// NewTreeView creates a new tree view.
func NewTreeView(manager *session.Manager) *TreeView {
	tree := components.NewSessionTree()
	log := components.NewLogViewport()

	tree.SetFocused(true)
	log.SetFocused(false)

	return &TreeView{
		tree:     tree,
		log:      log,
		focus:    FocusTree,
		manager:  manager,
		renderer: NewRenderer(NewStyles()),
	}
}

// SetSize sets the dimensions of the tree view.
func (tv *TreeView) SetSize(width, height int) {
	tv.width = width
	tv.height = height - 1 // Leave room for help line.

	tv.updateLayout()
}

func (tv *TreeView) updateLayout() {
	if tv.treeHidden {
		tv.tree.SetSize(0, tv.height)
		tv.log.SetSize(tv.width, tv.height)
		return
	}

	// Tree takes 30% of width (min 20, max 40).
	treeWidth := tv.width * 30 / 100
	if treeWidth < 20 {
		treeWidth = 20
	}
	if treeWidth > 40 {
		treeWidth = 40
	}

	logWidth := tv.width - treeWidth

	tv.tree.SetSize(treeWidth, tv.height)
	tv.log.SetSize(logWidth, tv.height)
}

// Update handles messages for tree view.
func (tv *TreeView) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		// Pass to focused component.
		if tv.focus == FocusTree {
			return tv.tree.Update(msg)
		}

		return tv.log.Update(msg)
	}

	switch keyMsg.String() {
	case "enter":
		if tv.focus == FocusTree {
			tv.setFocus(FocusLog)

			return nil
		}
	case "esc":
		if tv.focus == FocusLog {
			if tv.treeHidden {
				tv.treeHidden = false
				tv.updateLayout()
			}
			tv.setFocus(FocusTree)

			return nil
		}
	case "f":
		if tv.focus == FocusLog {
			tv.treeHidden = !tv.treeHidden
			tv.updateLayout()

			return nil
		}
	case "j", "down":
		if tv.focus == FocusTree {
			tv.tree.MoveDown()
			tv.updateLogSession()
		} else {
			tv.log.ScrollDown()
		}

		return nil
	case "k", "up":
		if tv.focus == FocusTree {
			tv.tree.MoveUp()
			tv.updateLogSession()
		} else {
			tv.log.ScrollUp()
		}

		return nil
	}

	// Pass to focused component.
	if tv.focus == FocusTree {
		return tv.tree.Update(keyMsg)
	}

	return tv.log.Update(keyMsg)
}

// View renders the tree view.
func (tv *TreeView) View() string {
	var main string
	if tv.treeHidden {
		main = tv.log.View()
	} else {
		treeView := tv.tree.View()
		logView := tv.log.View()
		main = lipgloss.JoinHorizontal(lipgloss.Top, treeView, logView)
	}

	// Help line.
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	var help string
	if tv.focus == FocusTree {
		help = helpStyle.Render("j/k: select | Enter: view logs | t: panel mode | q: quit")
	} else if tv.treeHidden {
		help = helpStyle.Render("j/k: scroll | f: show tree | Esc: back to tree | t: panel mode | q: quit")
	} else {
		help = helpStyle.Render("j/k: scroll | f: fullscreen | Esc: back to tree | t: panel mode | q: quit")
	}

	return lipgloss.JoinVertical(lipgloss.Left, main, help)
}

// RefreshSessions updates the session tree from the manager.
func (tv *TreeView) RefreshSessions() {
	nodes := tv.manager.GetSessionTree()
	tv.tree.SetSessionTree(nodes)
	tv.updateLogSession()
}

// RefreshLog refreshes the log viewport content.
func (tv *TreeView) RefreshLog() {
	tv.log.Refresh()
}

func (tv *TreeView) setFocus(focus Focus) {
	tv.focus = focus
	tv.tree.SetFocused(focus == FocusTree)
	tv.log.SetFocused(focus == FocusLog)
}

func (tv *TreeView) updateLogSession() {
	sess := tv.tree.SelectedSession()
	tv.log.SetSession(sess)
}

// GetFocus returns the current focus.
func (tv *TreeView) GetFocus() Focus {
	return tv.focus
}
