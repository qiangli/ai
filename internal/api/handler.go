package api

import (
	"context"
	"fmt"
	"strings"
)

type Request struct {
	Agent      string `json:"agent"`
	Subcommand string `json:"subcommand"`

	Message string `json:"message"`
	Content string `json:"content"`
}

func (r *Request) IsEmpty() bool {
	return r.Message == "" && r.Content == ""
}

func (r *Request) Input() string {
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

// Clip returns a clipped version of the content.
// This is intended for "smart" agents to make decisions based on user inputs.
func (r *Request) Clip() string {
	switch {
	case r.Message == "" && r.Content == "":
		return ""
	case r.Message == "":
		return clipText(r.Content, 500)
	case r.Content == "":
		return r.Message
	default:
		return fmt.Sprintf("###\n%s\n###\n%s", r.Message, clipText(r.Content, 500))
	}
}

// clipText truncates the input text to no more than the specified maximum length.
func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
	}
	return text
}

type Response struct {
	Agent string `json:"agent"`

	Content string `json:"content"`
}

type HandlerNext = func(context.Context, *Request) (*Response, error)

type Handler = func(context.Context, *Request, HandlerNext) (*Response, error)

type Agent interface {
	Handle(context.Context, *Request, HandlerNext) (*Response, error)
}

type Action = func(context.Context, string) (string, error)
