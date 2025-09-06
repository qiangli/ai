package api

import (
	"fmt"
	// "net/http"
	"os"
	"strings"

	"github.com/openai/openai-go"
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
	Agent   string `json:"agent"`
	Command string `json:"command"`

	// query - command line args
	Message string `json:"message"`

	// query - clipboard/stdin/editor
	Content string `json:"content"`

	// TODO deprecate
	Template string `json:"template"`

	Files []string `json:"files"`

	Extra map[string]any `json:"extra"`

	// // cached file contents
	// Messages []*Message `json:"-"`
}

func (r *UserInput) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent: %s/%s\n", r.Agent, r.Command))
	sb.WriteString(fmt.Sprintf("Message#: %v\n", len(r.Message)))
	sb.WriteString(fmt.Sprintf("Content#: %v\n", len(r.Content)))
	sb.WriteString(fmt.Sprintf("Intent: %s\n", r.Intent()))
	sb.WriteString(fmt.Sprintf("Files: %v\n", r.Files))
	// sb.WriteString(fmt.Sprintf("Messages#: %v\n", len(r.Messages)))
	// for _, v := range r.Messages {
	// 	sb.WriteString(fmt.Sprintf("ContentType: %s Content#: %v\n", v.ContentType, len(v.Content)))
	// }

	return sb.String()
}

// No user input.
func (r *UserInput) IsEmpty() bool {
	return r.Message == "" && r.Content == "" && len(r.Files) == 0
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

func (r *UserInput) FileContent() (string, error) {
	var b strings.Builder
	if len(r.Files) > 0 {
		for _, f := range r.Files {
			b.WriteString("\n### " + f + " ###\n")
			c, err := os.ReadFile(f)
			if err != nil {
				return "", err

			}
			b.WriteString(string(c))
		}
	}
	return b.String(), nil
}

// func (r *UserInput) FileMessages() ([]*Message, error) {
// 	if len(r.Messages) > 0 {
// 		return r.Messages, nil
// 	}

// 	var messages []*Message

// 	if len(r.Files) > 0 {
// 		for _, f := range r.Files {
// 			raw, err := os.ReadFile(f)
// 			if err != nil {
// 				return nil, err

// 			}
// 			mimeType := http.DetectContentType(raw)
// 			messages = append(messages, &Message{
// 				ContentType: mimeType,
// 				Content:     string(raw),
// 			})
// 		}
// 	}
// 	r.Messages = messages
// 	return messages, nil
// }

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
