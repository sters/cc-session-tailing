package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/sters/cc-session-tailing/internal/parser"
	"github.com/sters/cc-session-tailing/internal/session"
)

// Styles holds all panel styles.
type Styles struct {
	PanelBorder    lipgloss.Style
	HeaderStyle    lipgloss.Style
	ThinkStyle     lipgloss.Style
	TextStyle      lipgloss.Style
	ToolStyle      lipgloss.Style
	ToolInputStyle lipgloss.Style
	UserStyle      lipgloss.Style
	LabelStyle     lipgloss.Style
	EmptyStyle     lipgloss.Style
	HelpStyle      lipgloss.Style
}

// NewStyles creates a new Styles instance.
func NewStyles() *Styles {
	return &Styles{
		PanelBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")),
		HeaderStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("235")).
			Padding(0, 1),
		ThinkStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true),
		TextStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")),
		ToolStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
		ToolInputStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")),
		UserStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Bold(true),
		LabelStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		EmptyStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true),
		HelpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1),
	}
}

// Renderer handles panel rendering with styles.
type Renderer struct {
	styles *Styles
}

// NewRenderer creates a new Renderer.
func NewRenderer(styles *Styles) *Renderer {
	return &Renderer{styles: styles}
}

// RenderPanel renders a single panel.
func (r *Renderer) RenderPanel(sess *session.Session, width, height, scrollPos int) string {
	if sess == nil {
		return r.renderEmptyPanel(width, height)
	}

	// Calculate inner dimensions.
	innerWidth := width - 2   // border
	innerHeight := height - 2 // border

	if innerWidth < 10 || innerHeight < 3 {
		return ""
	}

	// Render header (account for scrollbar width).
	header := r.renderHeader(sess, innerWidth-1)

	headerHeight := lipgloss.Height(header)

	// Render body (account for scrollbar width).
	bodyHeight := innerHeight - headerHeight
	bodyWidth := innerWidth - 1 // Reserve space for scrollbar
	body, totalLines := r.renderBodyWithInfo(sess, bodyWidth, bodyHeight, scrollPos)

	// Render scrollbar.
	scrollbar := r.renderScrollbar(bodyHeight, totalLines, bodyHeight, scrollPos)

	// Combine body with scrollbar.
	bodyWithScrollbar := lipgloss.JoinHorizontal(lipgloss.Top, body, scrollbar)

	// Combine header and body.
	content := lipgloss.JoinVertical(lipgloss.Left, header, bodyWithScrollbar)

	return r.styles.PanelBorder.Width(innerWidth).Height(innerHeight).Render(content)
}

func (r *Renderer) renderEmptyPanel(width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2

	if innerWidth < 10 || innerHeight < 3 {
		return ""
	}

	content := r.styles.EmptyStyle.Render("Waiting for session...")

	return r.styles.PanelBorder.Width(innerWidth).Height(innerHeight).Render(
		lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, content),
	)
}

func (r *Renderer) renderHeader(sess *session.Session, width int) string {
	// Shorten session ID if needed.
	id := sess.ID
	if runewidth.StringWidth(id) > width-4 {
		id = runewidth.Truncate(id, width-7, "...")
	}

	prefix := ""
	if sess.IsSubagent {
		prefix = "[SUB] "
	}

	return r.styles.HeaderStyle.Width(width).Render(prefix + id)
}

func (r *Renderer) renderBody(sess *session.Session, width, height, scrollPos int) string {
	body, _ := r.renderBodyWithInfo(sess, width, height, scrollPos)
	return body
}

func (r *Renderer) renderBodyWithInfo(sess *session.Session, width, height, scrollPos int) (string, int) {
	if len(sess.Messages) == 0 {
		emptyLine := r.styles.EmptyStyle.Render("No messages yet...")
		// Pad to fixed width.
		padded := lipgloss.NewStyle().Width(width).Render(emptyLine)
		return padded, 0
	}

	lines := make([]string, 0, len(sess.Messages)*3)

	// Render messages from oldest to newest.
	for i := range sess.Messages {
		msg := sess.Messages[i]
		msgLines := r.renderMessage(msg, width)
		lines = append(lines, msgLines...)
	}

	// Calculate visible window.
	// scrollPos = 0 means show newest (bottom of content).
	// scrollPos > 0 means scroll up to see older content.
	totalLines := len(lines)

	// Clamp scrollPos to valid range.
	maxScroll := totalLines - height
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scrollPos > maxScroll {
		scrollPos = maxScroll
	}

	// Calculate the end position (newest line to show).
	endPos := totalLines - scrollPos
	if endPos < height {
		endPos = height
	}
	if endPos > totalLines {
		endPos = totalLines
	}

	// Calculate the start position.
	startPos := endPos - height
	if startPos < 0 {
		startPos = 0
	}

	// Extract visible lines and pad each to fixed width.
	visibleLines := lines[startPos:endPos]
	paddedLines := make([]string, len(visibleLines))
	lineStyle := lipgloss.NewStyle().Width(width)
	for i, line := range visibleLines {
		paddedLines[i] = lineStyle.Render(line)
	}

	return strings.Join(paddedLines, "\n"), totalLines
}

// renderScrollbar renders a scrollbar indicator.
func (r *Renderer) renderScrollbar(height, totalLines, visibleLines, scrollPos int) string {
	scrollbarStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	thumbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	// If all content fits, show empty track.
	if totalLines <= visibleLines {
		var lines []string
		for i := 0; i < height; i++ {
			lines = append(lines, scrollbarStyle.Render("│"))
		}
		return strings.Join(lines, "\n")
	}

	// Calculate thumb size and position.
	thumbHeight := max(1, height*visibleLines/totalLines)
	scrollableRange := totalLines - visibleLines

	// Clamp scrollPos to valid range.
	if scrollPos > scrollableRange {
		scrollPos = scrollableRange
	}
	if scrollPos < 0 {
		scrollPos = 0
	}

	thumbPos := 0
	if scrollableRange > 0 {
		// scrollPos=0 means at bottom, scrollPos=scrollableRange means at top
		// We want thumbPos=0 when at top, thumbPos=height-thumbHeight when at bottom
		thumbPos = int(float64(scrollableRange-scrollPos) / float64(scrollableRange) * float64(height-thumbHeight))
	}

	// Clamp thumbPos to valid range.
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > height-thumbHeight {
		thumbPos = height - thumbHeight
	}

	var lines []string
	for i := 0; i < height; i++ {
		if i >= thumbPos && i < thumbPos+thumbHeight {
			lines = append(lines, thumbStyle.Render("┃"))
		} else {
			lines = append(lines, scrollbarStyle.Render("│"))
		}
	}

	return strings.Join(lines, "\n")
}

func (r *Renderer) renderMessage(msg parser.Message, width int) []string {
	var lines []string

	for _, block := range msg.Message.Content {
		blockLines := r.renderContentBlock(block, width, msg.Type)
		lines = append(lines, blockLines...)
	}

	return lines
}

func (r *Renderer) renderContentBlock(block parser.ContentBlock, width int, msgType string) []string {
	var lines []string

	// Handle user messages.
	if msgType == "user" {
		if block.Type == "text" && block.Text != "" {
			label := r.styles.LabelStyle.Render("[USER] ")
			wrapped := wrapText(block.Text, width-7)
			for i, line := range wrapped {
				if i == 0 {
					lines = append(lines, label+r.styles.UserStyle.Render(line))
				} else {
					lines = append(lines, "       "+r.styles.UserStyle.Render(line))
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
			label := r.styles.LabelStyle.Render("[THINK] ")
			content := r.styles.ThinkStyle.Render(truncateText(text, width-8))
			lines = append(lines, label+content)
		}

	case "text":
		if block.Text != "" {
			label := r.styles.LabelStyle.Render("[TEXT] ")
			wrapped := wrapText(block.Text, width-7)
			for i, line := range wrapped {
				if i == 0 {
					lines = append(lines, label+r.styles.TextStyle.Render(line))
				} else {
					lines = append(lines, "       "+r.styles.TextStyle.Render(line))
				}
			}
		}

	case "tool_use":
		label := r.styles.LabelStyle.Render("[TOOL] ")
		toolName := r.styles.ToolStyle.Render(block.Name)
		lines = append(lines, label+toolName)

		// Show tool input.
		if block.Input != nil {
			inputStr := formatToolInput(block.Input, width-7)
			for _, line := range inputStr {
				lines = append(lines, "       "+r.styles.ToolInputStyle.Render(line))
			}
		}

	case "tool_result":
		label := r.styles.LabelStyle.Render("[RESULT] ")
		if block.Text != "" {
			content := truncateText(block.Text, width-9)
			lines = append(lines, label+r.styles.TextStyle.Render(content))
		}
	}

	return lines
}

func truncateText(text string, maxWidth int) string {
	// Remove newlines and truncate.
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
			// Find a good break point.
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

// findBreakPoint finds a good position to break text at the given width.
func findBreakPoint(text string, width int) int {
	// First, find the byte position that corresponds to the width.
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

	// Try to find a space to break at.
	for i := bytePos; i > 0; i-- {
		if text[i] == ' ' {
			return i
		}
	}

	return bytePos
}

// formatToolInput formats tool input for display.
func formatToolInput(input any, maxWidth int) []string {
	if input == nil {
		return nil
	}

	// Try to format as JSON for readability.
	inputMap, ok := input.(map[string]any)
	if !ok {
		str := fmt.Sprintf("%v", input)

		return []string{truncateText(str, maxWidth)}
	}

	// Sort keys for stable display order.
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
			// Truncate long strings.
			if len(v) > 50 {
				valueStr = v[:47] + "..."
			} else {
				valueStr = v
			}
			// Remove newlines for display.
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
