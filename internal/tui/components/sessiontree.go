package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/sters/cc-session-tailing/internal/session"
)

// TreeItem represents a flattened tree item for display.
type TreeItem struct {
	Session  *session.Session
	Depth    int
	HasChild bool
	IsLast   bool
}

// SessionTree is a hierarchical session tree display.
type SessionTree struct {
	items       []TreeItem
	nodes       []*session.Node
	selected    int
	width       int
	height      int
	focused     bool
	offset      int             // scroll offset
	highlighted map[string]bool // session IDs that are currently highlighted
}

// NewSessionTree creates a new session tree.
func NewSessionTree() *SessionTree {
	return &SessionTree{
		focused:     true,
		highlighted: make(map[string]bool),
	}
}

// SetSize sets the dimensions of the tree.
func (t *SessionTree) SetSize(width, height int) {
	t.width = width
	t.height = height
}

// SetSessionTree updates the tree from Node structure.
func (t *SessionTree) SetSessionTree(nodes []*session.Node) {
	// Remember currently selected session ID to preserve focus.
	var selectedSessionID string
	if t.selected >= 0 && t.selected < len(t.items) {
		selectedSessionID = t.items[t.selected].Session.ID
	}

	// Preserve current display order if we already have nodes.
	if len(t.nodes) > 0 {
		nodes = t.preserveOrder(nodes)
	}

	t.nodes = nodes
	t.items = t.flattenTree(nodes, 0)

	// Try to find the previously selected session.
	if selectedSessionID != "" {
		for i, item := range t.items {
			if item.Session.ID == selectedSessionID {
				t.selected = i

				return
			}
		}
	}

	// Fall back to clamping selection if session not found.
	if t.selected >= len(t.items) && len(t.items) > 0 {
		t.selected = len(t.items) - 1
	}
}

// preserveOrder reorders nodes to match the current display order.
// Existing nodes keep their order, new nodes are appended at the end.
func (t *SessionTree) preserveOrder(newNodes []*session.Node) []*session.Node {
	// Build a map of new nodes by session ID.
	newNodeMap := make(map[string]*session.Node)
	for _, n := range newNodes {
		newNodeMap[n.Session.ID] = n
	}

	// Build result keeping existing order.
	result := make([]*session.Node, 0, len(newNodes))
	seen := make(map[string]bool)

	// First, add existing nodes in their current order (with updated data).
	for _, oldNode := range t.nodes {
		if newNode, exists := newNodeMap[oldNode.Session.ID]; exists {
			// Preserve children order recursively.
			newNode.Children = t.preserveChildOrder(oldNode.Children, newNode.Children)
			result = append(result, newNode)
			seen[oldNode.Session.ID] = true
		}
	}

	// Then, append any new nodes that weren't in the old tree.
	for _, n := range newNodes {
		if !seen[n.Session.ID] {
			result = append(result, n)
		}
	}

	return result
}

// preserveChildOrder preserves the order of child nodes.
func (t *SessionTree) preserveChildOrder(oldChildren, newChildren []*session.Node) []*session.Node {
	if len(oldChildren) == 0 {
		return newChildren
	}

	// Build a map of new children by session ID.
	newChildMap := make(map[string]*session.Node)
	for _, n := range newChildren {
		newChildMap[n.Session.ID] = n
	}

	// Build result keeping existing order.
	result := make([]*session.Node, 0, len(newChildren))
	seen := make(map[string]bool)

	// First, add existing children in their current order.
	for _, oldChild := range oldChildren {
		if newChild, exists := newChildMap[oldChild.Session.ID]; exists {
			// Recursively preserve grandchildren order.
			newChild.Children = t.preserveChildOrder(oldChild.Children, newChild.Children)
			result = append(result, newChild)
			seen[oldChild.Session.ID] = true
		}
	}

	// Then, append any new children.
	for _, n := range newChildren {
		if !seen[n.Session.ID] {
			result = append(result, n)
		}
	}

	return result
}

// flattenTree converts the tree structure to a flat list for display.
func (t *SessionTree) flattenTree(nodes []*session.Node, depth int) []TreeItem {
	items := make([]TreeItem, 0, len(nodes))

	for i, node := range nodes {
		isLast := i == len(nodes)-1
		hasChild := len(node.Children) > 0

		items = append(items, TreeItem{
			Session:  node.Session,
			Depth:    depth,
			HasChild: hasChild,
			IsLast:   isLast,
		})

		if node.Expanded && hasChild {
			childItems := t.flattenTree(node.Children, depth+1)
			items = append(items, childItems...)
		}
	}

	return items
}

// SetFocused sets the focus state.
func (t *SessionTree) SetFocused(focused bool) {
	t.focused = focused
}

// IsFocused returns whether the tree is focused.
func (t *SessionTree) IsFocused() bool {
	return t.focused
}

// SelectedSession returns the currently selected session.
func (t *SessionTree) SelectedSession() *session.Session {
	if len(t.items) == 0 || t.selected < 0 || t.selected >= len(t.items) {
		return nil
	}

	return t.items[t.selected].Session
}

// Update handles messages.
func (t *SessionTree) Update(_ tea.Msg) tea.Cmd {
	if !t.focused {
		return nil
	}

	return nil
}

// View renders the tree.
func (t *SessionTree) View() string {
	borderColor := lipgloss.Color("240")
	if t.focused {
		borderColor = lipgloss.Color("212")
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(t.width - 2).
		Height(t.height - 2)

	if len(t.items) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

		return borderStyle.Render(emptyStyle.Render("No sessions"))
	}

	// Calculate visible area.
	contentHeight := t.height - 4 // borders
	t.adjustScroll(contentHeight)

	// Render visible items.
	var lines []string
	endIdx := t.offset + contentHeight
	if endIdx > len(t.items) {
		endIdx = len(t.items)
	}

	for i := t.offset; i < endIdx; i++ {
		line := t.renderItem(i)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")

	return borderStyle.Render(content)
}

func (t *SessionTree) adjustScroll(visibleHeight int) {
	// Ensure selected item is visible.
	if t.selected < t.offset {
		t.offset = t.selected
	}
	if t.selected >= t.offset+visibleHeight {
		t.offset = t.selected - visibleHeight + 1
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

func (t *SessionTree) renderItem(idx int) string {
	item := t.items[idx]
	isSelected := idx == t.selected
	isHighlighted := t.highlighted[item.Session.ID]

	// Build prefix for tree structure.
	prefix := strings.Repeat("  ", item.Depth)
	if item.Depth > 0 {
		if item.IsLast {
			prefix = strings.Repeat("  ", item.Depth-1) + "└─"
		} else {
			prefix = strings.Repeat("  ", item.Depth-1) + "├─"
		}
	}

	// Session name.
	name := item.Session.ID
	if item.Session.IsSubagent {
		// Extract just the agent part for subagents.
		parts := strings.Split(name, "/")
		if len(parts) > 1 {
			name = parts[len(parts)-1]
		}
	}

	// Child indicator.
	childIndicator := ""
	if item.HasChild {
		childIndicator = " ▶"
	}

	// Message count.
	msgCount := len(item.Session.Messages)
	countStr := fmt.Sprintf(" (%d)", msgCount)

	// Update indicator for highlighted sessions.
	updateIndicator := ""
	if isHighlighted && !isSelected {
		updateIndicator = " ●"
	}

	// Calculate available width.
	availWidth := t.width - 6 - runewidth.StringWidth(prefix) - runewidth.StringWidth(childIndicator) - runewidth.StringWidth(countStr) - runewidth.StringWidth(updateIndicator)
	if availWidth < 10 {
		availWidth = 10
	}

	// Truncate name if needed.
	if runewidth.StringWidth(name) > availWidth {
		name = runewidth.Truncate(name, availWidth-3, "...")
	}

	// Build the line.
	line := prefix + name + childIndicator + countStr

	// Apply styles.
	if isSelected {
		selectedStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("212")).
			Foreground(lipgloss.Color("235")).
			Bold(true).
			Width(t.width - 4)

		return selectedStyle.Render(line)
	}

	if isHighlighted {
		// Highlighted style - yellow/orange background flash effect.
		highlightStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("220")). // Yellow background
			Foreground(lipgloss.Color("235")). // Dark text
			Bold(true).
			Width(t.width - 4)

		return highlightStyle.Render(line + updateIndicator)
	}

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(t.width - 4)

	if item.Session.IsSubagent {
		normalStyle = normalStyle.Foreground(lipgloss.Color("243"))
	}

	return normalStyle.Render(line)
}

// MoveUp moves selection up.
func (t *SessionTree) MoveUp() {
	if t.selected > 0 {
		t.selected--
	}
}

// MoveDown moves selection down.
func (t *SessionTree) MoveDown() {
	if t.selected < len(t.items)-1 {
		t.selected++
	}
}

// MoveToChild moves to the first child of the selected session.
func (t *SessionTree) MoveToChild() bool {
	if t.selected < 0 || t.selected >= len(t.items) {
		return false
	}

	current := t.items[t.selected]
	if !current.HasChild {
		return false
	}

	// Find the first child (next item with depth+1).
	for i := t.selected + 1; i < len(t.items); i++ {
		if t.items[i].Depth == current.Depth+1 {
			t.selected = i

			return true
		}
		if t.items[i].Depth <= current.Depth {
			break
		}
	}

	return false
}

// ResetSelection resets the selection to the first item.
func (t *SessionTree) ResetSelection() {
	t.selected = 0
	t.offset = 0
}

// MoveToParent moves to the parent of the selected session.
func (t *SessionTree) MoveToParent() bool {
	if t.selected < 0 || t.selected >= len(t.items) {
		return false
	}

	current := t.items[t.selected]
	if current.Depth == 0 {
		return false
	}

	// Find the parent (previous item with depth-1).
	for i := t.selected - 1; i >= 0; i-- {
		if t.items[i].Depth == current.Depth-1 {
			t.selected = i

			return true
		}
	}

	return false
}

// HasChildren returns whether the selected session has children.
func (t *SessionTree) HasChildren() bool {
	if t.selected < 0 || t.selected >= len(t.items) {
		return false
	}

	return t.items[t.selected].HasChild
}

// HasParent returns whether the selected session has a parent.
func (t *SessionTree) HasParent() bool {
	if t.selected < 0 || t.selected >= len(t.items) {
		return false
	}

	return t.items[t.selected].Depth > 0
}

// SetHighlighted sets the highlighted session IDs.
func (t *SessionTree) SetHighlighted(sessionIDs map[string]bool) {
	t.highlighted = sessionIDs
}

// ClearHighlighted clears all highlighted session IDs.
func (t *SessionTree) ClearHighlighted() {
	t.highlighted = make(map[string]bool)
}

// HasHighlighted returns whether there are any highlighted sessions.
func (t *SessionTree) HasHighlighted() bool {
	return len(t.highlighted) > 0
}
