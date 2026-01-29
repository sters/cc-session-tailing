package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ContentBlock represents a single content block in a message.
type ContentBlock struct {
	Type     string `json:"type"` // "thinking", "text", "tool_use", "tool_result"
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
	Name     string `json:"name,omitempty"`  // tool name
	Input    any    `json:"input,omitempty"` // tool input
}

// MessageContent represents the content of a message.
// Content can be either a string or an array of ContentBlocks.
type MessageContent struct {
	Content []ContentBlock
}

// UnmarshalJSON handles both string and array content.
func (m *MessageContent) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a struct with content array first.
	var structured struct {
		Content []ContentBlock `json:"content"`
	}
	if err := json.Unmarshal(data, &structured); err == nil && len(structured.Content) > 0 {
		m.Content = structured.Content

		return nil
	}

	// Try to unmarshal as a struct with content as string.
	var stringContent struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &stringContent); err == nil && stringContent.Content != "" {
		m.Content = []ContentBlock{{Type: "text", Text: stringContent.Content}}

		return nil
	}

	// Content might be at the top level as a string.
	var rawContent struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &rawContent); err == nil && len(rawContent.Content) > 0 {
		// Check if it's a string.
		var str string
		if err := json.Unmarshal(rawContent.Content, &str); err == nil {
			m.Content = []ContentBlock{{Type: "text", Text: str}}

			return nil
		}
		// Check if it's an array.
		var blocks []ContentBlock
		if err := json.Unmarshal(rawContent.Content, &blocks); err == nil {
			m.Content = blocks

			return nil
		}
	}

	return nil
}

// Message represents a single message in a JSONL file.
type Message struct {
	Type      string         `json:"type"` // "user", "assistant", "system"
	Message   MessageContent `json:"message"`
	AgentID   string         `json:"agentId,omitempty"`
	SessionID string         `json:"sessionId,omitempty"`
	Timestamp string         `json:"timestamp"`
}

// ParseFile reads a JSONL file and returns all messages.
func ParseFile(path string) ([]Message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	return Parse(file)
}

// Parse reads messages from a reader.
func Parse(r io.Reader) ([]Message, error) {
	var messages []Message
	scanner := bufio.NewScanner(r)

	// Set a larger buffer for long lines.
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip malformed lines.
			continue
		}
		messages = append(messages, msg)
	}

	if err := scanner.Err(); err != nil {
		return messages, fmt.Errorf("scanner error: %w", err)
	}

	return messages, nil
}

// ParseFromOffset reads messages from a file starting at a byte offset.
func ParseFromOffset(path string, offset int64) ([]Message, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, offset, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	if offset > 0 {
		if _, err := file.Seek(offset, io.SeekStart); err != nil {
			return nil, offset, fmt.Errorf("failed to seek to offset %d: %w", offset, err)
		}
	}

	messages, err := Parse(file)
	if err != nil {
		return messages, offset, err
	}

	// Get new offset.
	newOffset, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return messages, offset, fmt.Errorf("failed to get current offset: %w", err)
	}

	return messages, newOffset, nil
}
