package api

import (
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2"
)

const (
	ContentTypeText    = "text"
	ContentTypeB64JSON = string(openai.ImageGenerateParamsResponseFormatB64JSON)
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
	Agent string `json:"agent"`
	// Command string `json:"command"`

	// query - command line args
	Message string `json:"message"`

	// query - clipboard/stdin/editor
	Content string `json:"content"`

	// cached file contents
	Messages []*Message `json:"-"`
}

func (r *UserInput) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent: %s\n", r.Agent))
	sb.WriteString(fmt.Sprintf("Message#: %v\n", len(r.Message)))
	sb.WriteString(fmt.Sprintf("Content#: %v\n", len(r.Content)))
	sb.WriteString(fmt.Sprintf("Intent: %s\n", r.Intent()))

	return sb.String()
}

func (r *UserInput) Clone() *UserInput {
	return &UserInput{
		Agent:    r.Agent,
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
