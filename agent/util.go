package agent

import (
	"os"
	"strings"
	"sync"

	"github.com/mattn/go-isatty"
)

func OnceValue[T any](f func() T) func() T {
	var once sync.Once
	var valid bool
	var p any
	var result T
	return func() T {
		once.Do(func() {
			defer func() {
				p = recover()
				if !valid {
					panic(p)
				}
			}()
			result = f()
			valid = true
		})
		if !valid {
			panic(p)
		}
		return result
	}
}

// clipText truncates the input text to no more than the specified maximum length.
func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n..."
	}
	return text
}

var isInputTTY = OnceValue(func() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
})

var isOutputTTY = OnceValue(func() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
})
