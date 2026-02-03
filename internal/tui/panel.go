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
			Background(lipgloss.Color("235")),
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

	// Combine body lines with scrollbar lines manually for exact width control.
	bodyLines := strings.Split(body, "\n")
	scrollbarLines := strings.Split(scrollbar, "\n")

	var combinedLines []string
	for i := 0; i < len(bodyLines) || i < len(scrollbarLines); i++ {
		bodyLine := ""
		scrollLine := ""
		if i < len(bodyLines) {
			bodyLine = bodyLines[i]
		}
		if i < len(scrollbarLines) {
			scrollLine = scrollbarLines[i]
		}
		// Ensure body line is exactly bodyWidth.
		bodyLine = padToWidth(bodyLine, bodyWidth)
		combinedLines = append(combinedLines, bodyLine+scrollLine)
	}

	// Combine header and body.
	allLines := []string{header}
	allLines = append(allLines, combinedLines...)

	// Ensure we have exactly innerHeight lines.
	for len(allLines) < innerHeight {
		allLines = append(allLines, strings.Repeat(" ", innerWidth))
	}
	if len(allLines) > innerHeight {
		allLines = allLines[:innerHeight]
	}

	content := strings.Join(allLines, "\n")

	return r.styles.PanelBorder.Render(content)
}

func (r *Renderer) renderEmptyPanel(width, height int) string {
	innerWidth := width - 2
	innerHeight := height - 2

	if innerWidth < 10 || innerHeight < 3 {
		return ""
	}

	// Build centered content manually with exact width.
	text := "Waiting for session..."
	textWidth := runewidth.StringWidth(text)

	// Calculate padding for centering.
	leftPad := (innerWidth - textWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	// Build lines with exact width.
	emptyLine := strings.Repeat(" ", innerWidth)
	lines := make([]string, 0, innerHeight)

	// Calculate vertical centering.
	topPad := (innerHeight - 1) / 2
	for range topPad {
		lines = append(lines, emptyLine)
	}

	// Add centered text line.
	centeredLine := strings.Repeat(" ", leftPad) + r.styles.EmptyStyle.Render(text)
	centeredLine = padToWidth(centeredLine, innerWidth)
	lines = append(lines, centeredLine)

	// Fill remaining lines.
	for len(lines) < innerHeight {
		lines = append(lines, emptyLine)
	}

	content := strings.Join(lines, "\n")

	return r.styles.PanelBorder.Render(content)
}

func (r *Renderer) renderHeader(sess *session.Session, width int) string {
	prefix := ""
	if sess.IsSubagent {
		prefix = "[SUB] "
	}

	// Calculate available width for ID (with 1 space padding on each side).
	availableWidth := width - 2 - runewidth.StringWidth(prefix)
	if availableWidth < 3 {
		availableWidth = 3
	}

	// Shorten session ID if needed.
	id := sess.ID
	if runewidth.StringWidth(id) > availableWidth {
		id = runewidth.Truncate(id, availableWidth-3, "...")
	}

	// Build content and pad to exact width.
	content := " " + prefix + id
	contentWidth := runewidth.StringWidth(content)
	if contentWidth < width {
		content += strings.Repeat(" ", width-contentWidth)
	}

	return r.styles.HeaderStyle.Render(content)
}

func (r *Renderer) renderBodyWithInfo(sess *session.Session, width, height, scrollPos int) (string, int) {
	if len(sess.Messages) == 0 {
		emptyLine := r.styles.EmptyStyle.Render("No messages yet...")
		// Pad to fixed width using runewidth.
		padded := padToWidth(emptyLine, width)

		return padded, 0
	}

	lines := make([]string, 0, len(sess.Messages)*3)

	// Render messages from oldest to newest.
	for i := range sess.Messages {
		msg := sess.Messages[i]
		msgLines := r.renderMessage(msg, width)
		lines = append(lines, msgLines...)
	}

	totalLines := len(lines)

	// Calculate visible window.
	// scrollPos = -1 means follow mode (show newest content at bottom).
	// scrollPos >= 0 means fixed mode (scrollPos is the start line index).
	startPos, endPos := calculateVisibleWindow(totalLines, height, scrollPos)

	// Extract visible lines and pad each to fixed width using runewidth.
	visibleLines := lines[startPos:endPos]
	paddedLines := make([]string, len(visibleLines))
	for i, line := range visibleLines {
		paddedLines[i] = padToWidth(line, width)
	}

	return strings.Join(paddedLines, "\n"), totalLines
}

// calculateVisibleWindow calculates the start and end positions for visible content.
// scrollPos = -1 means follow mode, >= 0 means fixed start line.
func calculateVisibleWindow(totalLines, height, scrollPos int) (int, int) {
	var startPos, endPos int

	if scrollPos < 0 {
		// Follow mode: show the newest content.
		endPos = totalLines
		startPos = endPos - height
	} else {
		// Fixed mode: start from the specified line.
		startPos = scrollPos
		endPos = startPos + height
	}

	// Clamp to valid range.
	startPos = max(0, min(startPos, totalLines-height))
	endPos = max(startPos, min(endPos, totalLines))

	return startPos, endPos
}

// padToWidth pads a string to exact width.
// Uses lipgloss.Width to correctly handle ANSI escape sequences.
func padToWidth(s string, width int) string {
	currentWidth := lipgloss.Width(s)
	if currentWidth > width {
		// Need to truncate. Handle ANSI escape sequences.
		s = truncateWithANSI(s, width)
		currentWidth = lipgloss.Width(s)
	}
	if currentWidth < width {
		return s + strings.Repeat(" ", width-currentWidth)
	}

	return s
}

// truncateWithANSI truncates a string with ANSI escape sequences to fit within width.
func truncateWithANSI(s string, width int) string {
	var result strings.Builder
	currentWidth := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)

			continue
		}

		if inEscape {
			result.WriteRune(r)
			// End of escape sequence.
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}

			continue
		}

		rw := runewidth.RuneWidth(r)
		if currentWidth+rw > width {
			break
		}
		result.WriteRune(r)
		currentWidth += rw
	}

	// Reset any open ANSI sequences.
	result.WriteString("\x1b[0m")

	return result.String()
}

// renderScrollbar renders a scrollbar indicator.
// scrollPos: -1 = follow mode (at bottom), >= 0 = fixed start line.
func (r *Renderer) renderScrollbar(height, totalLines, visibleLines, scrollPos int) string {
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

	// Calculate the actual start line for scrollbar position.
	var startLine int
	if scrollPos < 0 {
		// Follow mode: at the bottom.
		startLine = scrollableRange
	} else {
		startLine = scrollPos
		if startLine > scrollableRange {
			startLine = scrollableRange
		}
	}

	thumbPos := 0
	if scrollableRange > 0 {
		// startLine=0 means at top, startLine=scrollableRange means at bottom
		// We want thumbPos=0 when at top, thumbPos=height-thumbHeight when at bottom
		thumbPos = int(float64(startLine) / float64(scrollableRange) * float64(height-thumbHeight))
	}

	// Clamp thumbPos to valid range.
	if thumbPos < 0 {
		thumbPos = 0
	}
	if thumbPos > height-thumbHeight {
		thumbPos = height - thumbHeight
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

	// Helper to ensure line fits within width (truncate before style application).
	ensureWidth := func(text string, maxWidth int) string {
		if runewidth.StringWidth(text) > maxWidth {
			return runewidth.Truncate(text, maxWidth, "")
		}

		return text
	}

	// Handle user messages.
	if msgType == "user" { //nolint:nestif // user message handling has necessary nested conditions
		if block.Type == "text" && block.Text != "" {
			label := r.styles.LabelStyle.Render("[USER] ")
			labelWidth := lipgloss.Width(label)
			indent := strings.Repeat(" ", labelWidth)
			contentWidth := width - labelWidth
			if contentWidth < 1 {
				contentWidth = 1
			}
			wrapped := wrapText(block.Text, contentWidth)
			for i, line := range wrapped {
				line = ensureWidth(line, contentWidth)
				if i == 0 {
					lines = append(lines, label+r.styles.UserStyle.Render(line))
				} else {
					lines = append(lines, indent+r.styles.UserStyle.Render(line))
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
			labelWidth := lipgloss.Width(label)
			contentWidth := width - labelWidth
			if contentWidth < 1 {
				contentWidth = 1
			}
			content := r.styles.ThinkStyle.Render(truncateText(text, contentWidth))
			lines = append(lines, label+content)
		}

	case "text":
		if block.Text != "" {
			label := r.styles.LabelStyle.Render("[TEXT] ")
			labelWidth := lipgloss.Width(label)
			indent := strings.Repeat(" ", labelWidth)
			contentWidth := width - labelWidth
			if contentWidth < 1 {
				contentWidth = 1
			}
			wrapped := wrapText(block.Text, contentWidth)
			for i, line := range wrapped {
				line = ensureWidth(line, contentWidth)
				if i == 0 {
					lines = append(lines, label+r.styles.TextStyle.Render(line))
				} else {
					lines = append(lines, indent+r.styles.TextStyle.Render(line))
				}
			}
		}

	case "tool_use":
		label := r.styles.LabelStyle.Render("[TOOL] ")
		labelWidth := lipgloss.Width(label)
		indent := strings.Repeat(" ", labelWidth)
		contentWidth := width - labelWidth
		if contentWidth < 1 {
			contentWidth = 1
		}
		// Truncate tool name if needed.
		toolNameTrunc := truncateText(block.Name, contentWidth)
		toolName := r.styles.ToolStyle.Render(toolNameTrunc)
		lines = append(lines, label+toolName)

		// Show tool input.
		if block.Input != nil {
			inputStr := formatToolInput(block.Input, contentWidth)
			for _, line := range inputStr {
				line = ensureWidth(line, contentWidth)
				lines = append(lines, indent+r.styles.ToolInputStyle.Render(line))
			}
		}

	case "tool_result":
		label := r.styles.LabelStyle.Render("[RESULT] ")
		labelWidth := lipgloss.Width(label)
		contentWidth := width - labelWidth
		if contentWidth < 1 {
			contentWidth = 1
		}
		if block.Text != "" {
			content := truncateText(block.Text, contentWidth)
			lines = append(lines, label+r.styles.TextStyle.Render(content))
		}
	}

	return lines
}

func truncateText(text string, maxWidth int) string {
	if maxWidth < 4 {
		maxWidth = 4
	}

	// Remove newlines and tabs, then truncate.
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "\t", " ")

	if runewidth.StringWidth(text) <= maxWidth {
		return text
	}

	// Truncate with "..." suffix, ensuring result fits in maxWidth.
	truncated := runewidth.Truncate(text, maxWidth, "...")

	// Double-check: if still over maxWidth, truncate more aggressively.
	for runewidth.StringWidth(truncated) > maxWidth {
		runes := []rune(strings.TrimSuffix(truncated, "..."))
		if len(runes) <= 1 {
			return "..."
		}
		truncated = string(runes[:len(runes)-1]) + "..."
	}

	return truncated
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
			if breakAt <= 0 {
				// Safety: avoid infinite loop.
				breakAt = 1
			}
			line := para[:breakAt]
			// Double-check: ensure line fits in width.
			if runewidth.StringWidth(line) > width {
				line = runewidth.Truncate(line, width, "")
			}
			lines = append(lines, line)
			para = strings.TrimLeft(para[breakAt:], " ")
		}
		if para != "" {
			// Final check on remaining text.
			if runewidth.StringWidth(para) > width {
				para = runewidth.Truncate(para, width, "")
			}
			lines = append(lines, para)
		}
	}

	return lines
}

// findBreakPoint finds a good position to break text at the given width.
func findBreakPoint(text string, width int) int {
	// Build a slice of (bytePos, runeWidth) for each rune.
	type runeInfo struct {
		bytePos  int
		endByte  int
		cumWidth int
		isSpace  bool
	}

	runes := make([]runeInfo, 0, len(text))
	currentWidth := 0

	for i, r := range text {
		rw := runewidth.RuneWidth(r)
		endByte := i + len(string(r))
		runes = append(runes, runeInfo{
			bytePos:  i,
			endByte:  endByte,
			cumWidth: currentWidth + rw,
			isSpace:  r == ' ',
		})
		currentWidth += rw
	}

	if len(runes) == 0 {
		return 0
	}

	// Find the last rune that fits within width.
	breakRuneIdx := len(runes) - 1
	for i, info := range runes {
		if info.cumWidth > width {
			if i > 0 {
				breakRuneIdx = i - 1
			} else {
				breakRuneIdx = 0
			}

			break
		}
	}

	// Try to find a space to break at (search backwards from breakRuneIdx).
	for i := breakRuneIdx; i >= 0; i-- {
		if runes[i].isSpace {
			return runes[i].bytePos
		}
	}

	// No space found, break at the calculated position.
	if breakRuneIdx < len(runes) {
		return runes[breakRuneIdx].endByte
	}

	return len(text)
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
		// Replace newlines first, then truncate.
		str = strings.ReplaceAll(str, "\n", "\\n")
		str = strings.ReplaceAll(str, "\r", "")

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
			// Replace newlines for display.
			valueStr = strings.ReplaceAll(v, "\n", "\\n")
			valueStr = strings.ReplaceAll(valueStr, "\r", "")
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				valueStr = fmt.Sprintf("%v", v)
			} else {
				valueStr = string(jsonBytes)
			}
		}

		// Format as "key: value" and truncate to maxWidth.
		line := fmt.Sprintf("%s: %s", key, valueStr)
		lines = append(lines, truncateText(line, maxWidth))
	}

	return lines
}
