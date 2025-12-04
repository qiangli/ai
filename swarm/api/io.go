package api

import (
// "maps"
// "strings"
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
	// cached media contents
	Messages []*Message `json:"-"`

	Message string

	// Arguments map[string]any `json:"arguments"`
}

// func (r *UserInput) String() string {
// 	if r.Arguments == nil {
// 		return ""
// 	}
// 	return ToString(r.Arguments["message"])
// }

// func (r *UserInput) Clone() *UserInput {
// 	ui := &UserInput{
// 		Messages: append([]*Message(nil), r.Messages...),
// 	}
// 	if r.Arguments != nil {
// 		args := make(map[string]any, len(r.Arguments))
// 		maps.Copy(args, r.Arguments)
// 		ui.Arguments = args
// 	}
// 	return ui
// }

// // No user input.
// func (r *UserInput) IsEmpty() bool {
// 	// return r.Message == "" && r.Content == "" && len(r.Files) == 0
// 	if r.Arguments == nil {
// 		return true
// 	}
// 	return r.Arguments["message"] == "" && r.Arguments["content"] == ""
// }

// // Text input from command line args, clipboard, stdin, or editor
// func (r *UserInput) Query() string {
// 	if r.Arguments == nil {
// 		return ""
// 	}
// 	q, ok := r.Arguments["message"]
// 	if ok {
// 		return ToString(q)
// 	}
// 	return ""
// }

// // Intent returns a clipped version of the query.
// // This is intended for "smart" agents to make decisions based on user inputs.
// func (r *UserInput) Intent() string {
// 	return clipText(r.Query(), 500)
// }

// // clipText truncates the input text to no more than the specified maximum length.
// func clipText(text string, maxLen int) string {
// 	if len(text) > maxLen {
// 		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
// 	}
// 	return text
// }

type Output struct {
	// Agent icon and name
	Display string `json:"display"`

	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

type IOFilter struct {
	Agent string
}
