package agent

import (
	"github.com/qiangli/ai/swarm/api"
)

const DefaultEditor = "vi"

const StdinRedirect = "-"

// clipboard redirection
const (
	// read from clipboard
	ClipinRedirect = "{"

	// read from clipboard and wait, allowing multiple copy
	// until Ctrl-D is entered
	ClipinRedirect2 = "{{"

	// write to clipboard, overwriting its content
	ClipoutRedirect = "}"

	// append to clipboard
	ClipoutRedirect2 = "}}"
)

// var Version = "0.0.1" // version of the ai binary

// parse special char sequence for stdin/clipboard
// they can:
// + be at the end of the args
// + be in any order
// + be multiple instances
func ParseSpecialChars(args []string) *api.InputConfig {
	var isStdin, isClipin, isClipWait, isClipout, isClipAppend bool

	if len(args) > 0 {
	loop:
		for i := len(args) - 1; i >= 0; i-- {
			lastArg := args[i]
			switch lastArg {
			case StdinRedirect:
				isStdin = true
			case ClipinRedirect:
				isClipin = true
			case ClipinRedirect2:
				isClipin = true
				isClipWait = true
			case ClipoutRedirect:
				isClipout = true
			case ClipoutRedirect2:
				isClipout = true
				isClipAppend = true
			default:
				// continue until a non special char
				break loop
			}
			args = args[:i+1]
		}
	}

	var cfg api.InputConfig

	cfg.Stdin = isStdin
	cfg.Clipin = isClipin
	cfg.ClipWait = isClipWait
	cfg.Clipout = isClipout
	cfg.ClipAppend = isClipAppend

	cfg.Args = args

	return &cfg
}
