package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

const clipMaxLen = 500

type Editor struct {
	editor string
}

func NewEditor(editor string) *Editor {
	return &Editor{
		editor: editor,
	}
}

func (e *Editor) Launch(content string) (string, error) {
	return LaunchEditor(e.editor, content)
}

// GetUserInput collects user input for the agent.
// It prefers a direct message from the command line flag (--message),
// otherwise, it determines the input source (stdin, clipboard, editor)
// and collects input accordingly. It also
// attaches any provided files or template file if provided.
func GetUserInput(cfg *api.InputConfig, msg string) (*api.UserInput, error) {
	return getUserInput(cfg, msg, nil, nil, nil)
}

// user query: message and content
// cfg.Message is prepended to message collected from command line --message flag or the non flag/option args.
func getUserInput(cfg *api.InputConfig, message string, stdin io.Reader, clipper api.ClipboardProvider, editor api.EditorProvider) (*api.UserInput, error) {
	// collecting message content from various sources
	if clipper == nil {
		clipper = util.NewClipboard()
	}

	input, err := userInput(cfg, message, stdin, clipper)
	if err != nil {
		return nil, err
	}
	return input, nil
}

func userInput(
	cfg *api.InputConfig,
	message string,
	stdin io.Reader,
	clipboard api.ClipboardProvider,
) (*api.UserInput, error) {
	ctx := context.TODO()
	cat := func(a, b, sep string) string {
		if a != "" && b == "" {
			return a
		} else if a == "" && b != "" {
			return b
		} else if a != "" && b != "" {
			return a + sep + b
		}
		return ""
	}

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
				log.GetLogger(ctx).Infof("\n%s\n", clipText(v, 500))
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
	var content = cat(stdinData, clipinData, "\n")
	msg := cat(message, content, "\n###\n")
	return &api.UserInput{
		Message: msg,
	}, nil
}

func LaunchEditor(editor string, content string) (string, error) {
	tmpfile, err := os.CreateTemp("", "ai_*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	//
	if len(content) > 0 {
		if _, err := tmpfile.WriteString(content); err != nil {
			tmpfile.Close()
			return "", err
		}
		if err := tmpfile.Close(); err != nil {
			return "", err
		}
	}

	// open editor
	// support simple args for editor command line
	cmdArgs := strings.Fields(editor)
	var bin string
	var args []string
	bin = cmdArgs[0]
	if len(cmdArgs) > 1 {
		args = cmdArgs[1:]
	}
	args = append(args, tmpfile.Name())

	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	edited, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}
	return string(edited), nil
}

// PrintInput prints the user input
func PrintInput(ctx context.Context, cfg *api.AppConfig) {
	log.GetLogger(ctx).Debugf("UserInput:\n%+v\n", cfg)

	// query and files for info only
	var msg = clipText(cfg.Message, clipMaxLen)
	renderInputContent(ctx, msg)
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

func renderInputContent(ctx context.Context, content string) {
	md := util.Render(content)
	log.GetLogger(ctx).Infof("\n%s\n", md)
}

func ReadStdin() (string, error) {
	stdin := os.Stdin

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
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
