package swarm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// stdin
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

type InputConfig struct {
	// Message string
	Args []string

	Clipin     bool
	ClipWait   bool
	Clipout    bool
	ClipAppend bool
	Stdin      bool
}

// func GetInput(ctx context.Context, argv []string) (*InputConfig, error) {
// 	cfg := ParseSpecialChars(argv)

// 	in, err := GetUserInput(ctx, cfg)
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.Message = in.Message

// 	return cfg, nil
// }

// parse special char sequence for stdin/clipboard
// they can:
// + be at the end of the args
// + be in any order
// + be multiple instances
func ParseSpecialChars(args []string) *InputConfig {
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

	var cfg InputConfig

	cfg.Stdin = isStdin
	cfg.Clipin = isClipin
	cfg.ClipWait = isClipWait
	cfg.Clipout = isClipout
	cfg.ClipAppend = isClipAppend

	cfg.Args = args

	return &cfg
}

// GetUserInput collects user input for the agent.
// It prefers a direct message from the command line flag (--message),
// otherwise, it determines the input source (stdin, clipboard, editor)
// and collects input accordingly. It also
// attaches any provided files or template file if provided.
func GetUserInput(ctx context.Context, cfg *InputConfig) (*api.UserInput, error) {
	return getUserInput(ctx, cfg, nil, nil, nil)
}

// user query: message and content
// cfg.Message is prepended to message collected from command line --message flag or the non flag/option args.
func getUserInput(ctx context.Context, cfg *InputConfig, stdin io.Reader, clipper api.ClipboardProvider, editor api.EditorProvider) (*api.UserInput, error) {
	// collecting message content from various sources
	if clipper == nil {
		clipper = util.NewClipboard()
	}

	input, err := userInput(ctx, cfg, stdin, clipper)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func userInput(
	ctx context.Context,
	cfg *InputConfig,
	stdin io.Reader,
	clipboard api.ClipboardProvider,
) (*api.UserInput, error) {

	// stdin
	var stdinData string
	if cfg.Stdin {
		if stdin == nil {
			stdin = os.Stdin
		}
		log.GetLogger(ctx).Promptf("Please enter your input. Ctrl+D to send, Ctrl+C to cancel...\n")
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}

		stdinData = strings.TrimSpace(string(data))
	}

	// clipboard
	var clipinData string
	if cfg.Clipin {
		var data string
		if cfg.ClipWait {
			// paste-append from clipboard
			var pb []string
			for {
				if err := clipboard.Clear(); err != nil {
					return nil, err
				}

				log.GetLogger(ctx).Promptf("Awaiting clipboard content...\n")
				v, err := clipboard.Read()
				if err != nil {
					return nil, err
				}
				log.GetLogger(ctx).Infof("\n%s\n", clip(v, 500))
				send, err := pasteConfirm(ctx)
				// user canceled
				if err != nil {
					return nil, err
				}
				if send {
					pb = append(pb, v)
					data = strings.Join(pb, "\n")
					break
				}
				// continue appending
				pb = append(pb, v)
			}
		} else {
			if err := clipboard.Clear(); err != nil {
				return nil, err
			}

			log.GetLogger(ctx).Promptf("Awaiting clipboard content...\n")
			v, err := clipboard.Read()
			if err != nil {
				return nil, err
			}
			data = v
		}

		clipinData = strings.TrimSpace(data)
	}

	// update query
	var content = Cat(stdinData, clipinData, "\n")
	// msg := Cat(message, content, "\n###\n")
	return &api.UserInput{
		Message: content,
	}, nil
}

// pasteConfirm prompts the user to append, send, or cancel the input
// and returns true if the user chooses to send the input
func pasteConfirm(ctx context.Context) (bool, error) {
	ps := "Append to input, Send, or Cancel? [A/s/c] "
	choices := []string{"append", "send", "cancel"}
	defaultChoice := "append"

	answer, err := util.Confirm(ctx, ps, choices, defaultChoice, os.Stdin)
	if err != nil {
		return false, err
	}
	switch answer {
	case "send":
		return true, nil
	case "append":
		return false, nil
	}
	return false, fmt.Errorf("canceled")
}

func Cat(a, b, sep string) string {
	if a != "" && b == "" {
		return a
	} else if a == "" && b != "" {
		return b
	} else if a != "" && b != "" {
		return a + sep + b
	}
	return ""
}
