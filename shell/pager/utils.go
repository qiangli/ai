package pager

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// timoutContext setup a new context that times out if the given timeout is > 0.
func timeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if timeout == 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// LipglossPadding calculates how much padding a string is given by a style.
func LipglossPadding(style lipgloss.Style) (int, int) {
	render := style.Render(" ")
	before := strings.Index(render, " ")
	after := len(render) - len(" ") - before
	return before, after
}
