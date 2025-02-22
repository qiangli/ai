package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go"
)

const (
	ContentTypeText    = "text"
	ContentTypeB64JSON = string(openai.ImageGenerateParamsResponseFormatB64JSON)
)

type UserInput struct {
	Agent      string `json:"agent"`
	Subcommand string `json:"subcommand"`

	Message string `json:"message"`
	Content string `json:"content"`

	Template string `json:"template"`

	Files []string `json:"files"`

	Extra map[string]any `json:"extra"`
}

func (r *UserInput) IsEmpty() bool {
	return r.Message == "" && r.Content == "" && len(r.Files) == 0
}

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
