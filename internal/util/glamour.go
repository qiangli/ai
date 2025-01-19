package util

import (
	"github.com/charmbracelet/glamour"
)

func Render(text string) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
		glamour.WithEmoji(),
	)
	if err != nil {
		return text
	}
	styled, err := r.Render(text)
	if err != nil {
		return text
	}
	return styled
}
