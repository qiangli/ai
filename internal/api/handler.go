package api

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
)

type RequestType int

const (
	RequestText RequestType = iota
	RequestImage
	// RequestSound
	// RequestVision
)

type Request struct {
	Type RequestType `json:"type"`

	Agent      string `json:"agent"`
	Subcommand string `json:"subcommand"`

	Message string `json:"message"`
	Content string `json:"content"`

	Template string `json:"template"`

	Files []string `json:"files"`

	Extra map[string]any `json:"extra"`
}

func (r *Request) IsEmpty() bool {
	return r.Message == "" && r.Content == "" && len(r.Files) == 0
}

func (r *Request) Query() string {
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

func (r *Request) FileContent() (string, error) {
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

// Intent returns a clipped version of the query.
// This is intended for "smart" agents to make decisions based on user inputs.
func (r *Request) Intent() string {
	return clipText(r.Query(), 500)
}

// clipText truncates the input text to no more than the specified maximum length.
func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
	}
	return text
}

const (
	ContentTypeText    = "text"
	ContentTypeB64JSON = string(openai.ImageGenerateParamsResponseFormatB64JSON)
)

type Response struct {
	Agent string `json:"agent"`

	ContentType string `json:"content_type"`
	Content     string `json:"content"`
}

type HandlerNext = func(context.Context, *Request) (*Response, error)

type Handler = func(context.Context, *Request, HandlerNext) (*Response, error)

type Agent interface {
	Handle(context.Context, *Request, HandlerNext) (*Response, error)
}

type Action = func(context.Context, string) (string, error)
