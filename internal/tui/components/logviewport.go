package components

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/sters/cc-session-tailing/internal/parser"
	"github.com/sters/cc-session-tailing/internal/session"
)

// logStyles holds styles for log rendering.
type logStyles struct {
	thinkStyle     lipgloss.Style
	textStyle      lipgloss.Style
	toolStyle      lipgloss.Style
	toolInputStyle lipgloss.Style
	userStyle      lipgloss.Style
	labelStyle     lipgloss.Style
}

func newLogStyles() *logStyles {
	return &logStyles{
		thinkStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true),
		textStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		toolStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		toolInputStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),
		userStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Bold(true),
		labelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// LogViewport displays log content for a session.
type LogViewport struct {
	viewport viewport.Model
	session  *session.Session
	styles   *logStyles
	width    int
	height   int
	focused  bool
}

// NewLogViewport creates a new log viewport.
func NewLogViewport() *LogViewport {
	vp := viewport.New(0, 0)
	vp.SetContent("")

	return &LogViewport{
		viewport: vp,
		styles:   newLogStyles(),
	}
}

// SetSize sets the dimensions of the viewport.
func (l *LogViewport) SetSize(width, height int) {
	l.width = width
	l.height = height
	// Account for border and scrollbar.
	l.viewport.Width = width - 3   // border (2) + scrollbar (1)
	l.viewport.Height = height - 4 // border + header
}

// SetSession sets the session to display.
func (l *LogViewport) SetSession(s *session.Session) {
	l.session = s
	l.updateContent()
}

// SetFocused sets the focus state.
func (l *LogViewport) SetFocused(focused bool) {
	l.focused = focused
}

// IsFocused returns whether the viewport is focused.
func (l *LogViewport) IsFocused() bool {
	return l.focused
}

// Update handles messages.
func (l *LogViewport) Update(msg tea.Msg) tea.Cmd {
	if !l.focused {
		return nil
	}

	var cmd tea.Cmd
	l.viewport, cmd = l.viewport.Update(msg)

	return cmd
}

// View renders the viewport.
func (l *LogViewport) View() string {
	borderColor := lipgloss.Color("240")
	if l.focused {
		borderColor = lipgloss.Color("212")
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Width(l.width - 2).
		Height(l.height - 2)

	if l.session == nil {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		content := lipgloss.Place(
			l.width-4,
			l.height-4,
			lipgloss.Center,
			lipgloss.Center,
			emptyStyle.Render("Select a session to view logs"),
		)

		return borderStyle.Render(content)
	}

	// Header.
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(l.width - 5) // Account for scrollbar.

	prefix := ""
	if l.session.IsSubagent {
		prefix = "[SUB] "
	}
	header := headerStyle.Render(prefix + l.session.ID)

	// Render scrollbar.
	scrollbar := l.renderScrollbar()

	// Content with scrollbar.
	viewportContent := l.viewport.View()
	contentWithScrollbar := lipgloss.JoinHorizontal(lipgloss.Top, viewportContent, scrollbar)

	content := lipgloss.JoinVertical(lipgloss.Left, header, contentWithScrollbar)

	return borderStyle.Render(content)
}

// renderScrollbar renders a scrollbar indicator.
func (l *LogViewport) renderScrollbar() string {
	height := l.viewport.Height
	totalLines := l.viewport.TotalLineCount()
	visibleLines := l.viewport.Height
	yOffset := l.viewport.YOffset

	scrollbarStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// If all content fits, show empty track.
	if totalLines <= visibleLines {
		var lines []string
		for range height {
			lines = append(lines, scrollbarStyle.Render("│"))
		}

		return strings.Join(lines, "\n")
	}

	// Calculate thumb size and position.
	thumbHeight := max(1, height*visibleLines/totalLines)
	scrollableRange := totalLines - visibleLines
	thumbPos := 0
	if scrollableRange > 0 {
		thumbPos = int(float64(yOffset) / float64(scrollableRange) * float64(height-thumbHeight))
	}

	var lines []string
	for i := range height {
		if i >= thumbPos && i < thumbPos+thumbHeight {
			lines = append(lines, thumbStyle.Render("┃"))
		} else {
			lines = append(lines, scrollbarStyle.Render("│"))
		}
	}

	return strings.Join(lines, "\n")
}

// ScrollDown scrolls the viewport down.
func (l *LogViewport) ScrollDown() {
	l.viewport.ScrollDown(1)
}

// ScrollUp scrolls the viewport up.
func (l *LogViewport) ScrollUp() {
	l.viewport.ScrollUp(1)
}

// GotoBottom scrolls to the bottom of the content.
func (l *LogViewport) GotoBottom() {
	l.viewport.GotoBottom()
}

// updateContent updates the viewport content from the session.
func (l *LogViewport) updateContent() {
	if l.session == nil {
		l.viewport.SetContent("")

		return
	}

	contentWidth := l.width - 5 // border (2) + scrollbar (1) + padding (2)

	var lines []string
	for _, msg := range l.session.Messages {
		msgLines := l.renderMessage(msg, contentWidth)
		lines = append(lines, msgLines...)
	}

	content := strings.Join(lines, "\n")
	l.viewport.SetContent(content)
	l.viewport.GotoBottom()
}

func (l *LogViewport) renderMessage(msg parser.Message, width int) []string {
	var lines []string

	for _, block := range msg.Message.Content {
		blockLines := l.renderContentBlock(block, width, msg.Type)
		lines = append(lines, blockLines...)
	}

	return lines
}

func (l *LogViewport) renderContentBlock(block parser.ContentBlock, width int, msgType string) []string {
	var lines []string

	// Handle user messages.
	if msgType == "user" {
		if block.Type == "text" && block.Text != "" {
			label := l.styles.labelStyle.Render("[USER] ")
			wrapped := wrapText(block.Text, width-7)
			for i, line := range wrapped {
				if i == 0 {
					lines = append(lines, label+l.styles.userStyle.Render(line))
				} else {
					lines = append(lines, "       "+l.styles.userStyle.Render(line))
				}
			}
		}

		return lines
	}

	switch block.Type {
	case "thinking":
		text := block.Thinking
		if text == "" {
			text = block.Text
		}
		if text != "" {
			label := l.styles.labelStyle.Render("[THINK] ")
			content := l.styles.thinkStyle.Render(truncateText(text, width-8))
			lines = append(lines, label+content)
		}

	case "text":
		if block.Text != "" {
			label := l.styles.labelStyle.Render("[TEXT] ")
			wrapped := wrapText(block.Text, width-7)
			for i, line := range wrapped {
				if i == 0 {
					lines = append(lines, label+l.styles.textStyle.Render(line))
				} else {
					lines = append(lines, "       "+l.styles.textStyle.Render(line))
				}
			}
		}

	case "tool_use":
		label := l.styles.labelStyle.Render("[TOOL] ")
		toolName := l.styles.toolStyle.Render(block.Name)
		lines = append(lines, label+toolName)

		// Show tool input.
		if block.Input != nil {
			inputStr := formatToolInput(block.Input, width-7)
			for _, line := range inputStr {
				lines = append(lines, "       "+l.styles.toolInputStyle.Render(line))
			}
		}

	case "tool_result":
		label := l.styles.labelStyle.Render("[RESULT] ")
		if block.Text != "" {
			content := truncateText(block.Text, width-9)
			lines = append(lines, label+l.styles.textStyle.Render(content))
		}
	}

	return lines
}

// Refresh updates the content from the current session.
func (l *LogViewport) Refresh() {
	l.updateContent()
}

func truncateText(text string, maxWidth int) string {
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")

	if runewidth.StringWidth(text) > maxWidth {
		return runewidth.Truncate(text, maxWidth-3, "...")
	}

	return text
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	text = strings.ReplaceAll(text, "\r", "")
	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		if para == "" {
			lines = append(lines, "")

			continue
		}

		for runewidth.StringWidth(para) > width {
			breakAt := findBreakPoint(para, width)
			lines = append(lines, para[:breakAt])
			para = strings.TrimLeft(para[breakAt:], " ")
		}
		if para != "" {
			lines = append(lines, para)
		}
	}

	return lines
}

func findBreakPoint(text string, width int) int {
	currentWidth := 0
	bytePos := 0

	for i, r := range text {
		rw := runewidth.RuneWidth(r)
		if currentWidth+rw > width {
			bytePos = i

			break
		}
		currentWidth += rw
		bytePos = i + len(string(r))
	}

	for i := bytePos; i > 0; i-- {
		if text[i] == ' ' {
			return i
		}
	}

	return bytePos
}

func formatToolInput(input any, maxWidth int) []string {
	if input == nil {
		return nil
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		str := fmt.Sprintf("%v", input)

		return []string{truncateText(str, maxWidth)}
	}

	keys := make([]string, 0, len(inputMap))
	for key := range inputMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		value := inputMap[key]
		var valueStr string
		switch v := value.(type) {
		case string:
			if len(v) > 50 {
				valueStr = v[:47] + "..."
			} else {
				valueStr = v
			}
			valueStr = strings.ReplaceAll(valueStr, "\n", "\\n")
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				valueStr = fmt.Sprintf("%v", v)
			} else {
				valueStr = string(jsonBytes)
			}
			if len(valueStr) > 50 {
				valueStr = valueStr[:47] + "..."
			}
		}

		line := fmt.Sprintf("%s: %s", key, valueStr)
		lines = append(lines, truncateText(line, maxWidth))
	}

	return lines
}
