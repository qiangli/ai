package api

import (
	"fmt"
	"strings"
)

const (
	ContentTypeImageB64 = "img/base64"
)

type ClipboardProvider interface {
	Clear() error
	Read() (string, error)
	Get() (string, error)
	Write(string) error
	Append(string) error
}

type EditorProvider interface {
	Launch(string) (string, error)
}

type UserInput struct {
	// query - command line args
	Message string `json:"message"`

	// query - clipboard/stdin/editor
	// message is moved to content if editor is used
	Content string `json:"content"`

	// cached media contents
	Messages []*Message `json:"-"`

	// experimental
	Instruction *Instruction `json:"instruction"`
	Model       string       `json:"model"`
	Functions   []string     `json:"functions"`

	MaxTurns int `json:"max_turns"`
	MaxTime  int `json:"max_time"`
	// New        *bool `json:"new"`
	MaxHistory int `json:"max_history"`
	MaxSpan    int `json:"max_span"`
	// Model     string `json:"models"`
	Format   string `json:"format"`
	LogLevel string `json:"log_level"`

	Arguments map[string]any `json:"arguments"`
}

func (r *UserInput) String() string {
	return fmt.Sprintf("message: %v content: %v", len(r.Message), len(r.Content))
}

func (r *UserInput) Clone() *UserInput {
	return &UserInput{
		Message:  r.Message,
		Content:  r.Content,
		Messages: append([]*Message(nil), r.Messages...),
	}
}

// No user input.
func (r *UserInput) IsEmpty() bool {
	// return r.Message == "" && r.Content == "" && len(r.Files) == 0
	return r.Message == "" && r.Content == ""
}

// Text input from command line args, clipboard, stdin, or editor
func (r *UserInput) Query() string {
	switch {
	case r.Message == "" && r.Content == "":
		return ""
	case r.Message == "":
		return r.Content
	case r.Content == "":
		return r.Message
	default:
		return fmt.Sprintf("###\n%s\n###\n%s", r.Message, r.Content)
	}
}

// Intent returns a clipped version of the query.
// This is intended for "smart" agents to make decisions based on user inputs.
func (r *UserInput) Intent() string {
	return clipText(r.Query(), 500)
}

// clipText truncates the input text to no more than the specified maximum length.
func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
	}
	return text
}

type Output struct {
	// Agent icon and name
	Display string `json:"display"`

	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

type IOFilter struct {
	Agent string
}
