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

var (
	// Panel styles
	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	thinkStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	textStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	toolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	toolInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("117")).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
)

// RenderPanel renders a single panel.
func RenderPanel(sess *session.Session, width, height, scrollPos int) string {
	if sess == nil {
		return renderEmptyPanel(width, height)
	}

	// Calculate inner dimensions
	innerWidth := width - 2  // border
	innerHeight := height - 2 // border

	if innerWidth < 10 || innerHeight < 3 {
		return ""
	}

	// Render header
	header := renderHeader(sess, innerWidth)
	headerHeight := lipgloss.Height(header)

	// Render body
	bodyHeight := innerHeight - headerHeight
	body := renderBody(sess, innerWidth, bodyHeight, scrollPos)

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left, header, body)

	return panelBorder.Width(innerWidth).Height(innerHeight).Render(content)
}

func renderEmptyPanel(width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2

	if innerWidth < 10 || innerHeight < 3 {
		return ""
	}

	content := emptyStyle.Render("Waiting for session...")
	return panelBorder.Width(innerWidth).Height(innerHeight).Render(
		lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, content),
	)
}

func renderHeader(sess *session.Session, width int) string {
	// Shorten session ID if needed
	id := sess.ID
	if runewidth.StringWidth(id) > width-4 {
		id = runewidth.Truncate(id, width-7, "...")
	}

	prefix := ""
	if sess.IsSubagent {
		prefix = "[SUB] "
	}

	return headerStyle.Width(width).Render(prefix + id)
}

func renderBody(sess *session.Session, width, height, scrollPos int) string {
	if len(sess.Messages) == 0 {
		return emptyStyle.Render("No messages yet...")
	}

	var lines []string

	// Render messages from oldest to newest
	for i := 0; i < len(sess.Messages); i++ {
		msg := sess.Messages[i]
		msgLines := renderMessage(msg, width)
		lines = append(lines, msgLines...)
	}

	// Calculate visible window
	// scrollPos = 0 means show newest (bottom of content)
	// scrollPos > 0 means scroll up to see older content
	totalLines := len(lines)

	// Calculate the end position (newest line to show)
	endPos := totalLines - scrollPos
	if endPos < height {
		endPos = height
	}
	if endPos > totalLines {
		endPos = totalLines
	}

	// Calculate the start position
	startPos := endPos - height
	if startPos < 0 {
		startPos = 0
	}

	// Extract visible lines
	lines = lines[startPos:endPos]

	return strings.Join(lines, "\n")
}

func renderMessage(msg parser.Message, width int) []string {
	var lines []string

	for _, block := range msg.Message.Content {
		blockLines := renderContentBlock(block, width, msg.Type)
		lines = append(lines, blockLines...)
	}

	return lines
}

func renderContentBlock(block parser.ContentBlock, width int, msgType string) []string {
	var lines []string

	// Handle user messages
	if msgType == "user" {
		if block.Type == "text" && block.Text != "" {
			label := labelStyle.Render("[USER] ")
			wrapped := wrapText(block.Text, width-7)
			for i, line := range wrapped {
				if i == 0 {
					lines = append(lines, label+userStyle.Render(line))
				} else {
					lines = append(lines, "       "+userStyle.Render(line))
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
			label := labelStyle.Render("[THINK] ")
			content := thinkStyle.Render(truncateText(text, width-8))
			lines = append(lines, label+content)
		}

	case "text":
		if block.Text != "" {
			label := labelStyle.Render("[TEXT] ")
			wrapped := wrapText(block.Text, width-7)
			for i, line := range wrapped {
				if i == 0 {
					lines = append(lines, label+textStyle.Render(line))
				} else {
					lines = append(lines, "       "+textStyle.Render(line))
				}
			}
		}

	case "tool_use":
		label := labelStyle.Render("[TOOL] ")
		toolName := toolStyle.Render(block.Name)
		lines = append(lines, label+toolName)

		// Show tool input
		if block.Input != nil {
			inputStr := formatToolInput(block.Input, width-7)
			for _, line := range inputStr {
				lines = append(lines, "       "+toolInputStyle.Render(line))
			}
		}

	case "tool_result":
		label := labelStyle.Render("[RESULT] ")
		if block.Text != "" {
			content := truncateText(block.Text, width-9)
			lines = append(lines, label+textStyle.Render(content))
		}
	}

	return lines
}

func truncateText(text string, maxWidth int) string {
	// Remove newlines and truncate
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
			// Find a good break point
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
	// First, find the byte position that corresponds to the width
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

	// Try to find a space to break at
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

	// Try to format as JSON for readability
	inputMap, ok := input.(map[string]any)
	if !ok {
		str := fmt.Sprintf("%v", input)
		return []string{truncateText(str, maxWidth)}
	}

	// Sort keys for stable display order
	keys := make([]string, 0, len(inputMap))
	for key := range inputMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var lines []string
	for _, key := range keys {
		value := inputMap[key]
		var valueStr string
		switch v := value.(type) {
		case string:
			// Truncate long strings
			if len(v) > 50 {
				valueStr = v[:47] + "..."
			} else {
				valueStr = v
			}
			// Remove newlines for display
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
