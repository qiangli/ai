package agent

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

const clipMaxLen = 500

type ClipboardProvider = util.ClipboardProvider

type EditorProvider interface {
	Launch() (string, error)
}

type Editor struct {
	editor string
}

func NewEditor(editor string) EditorProvider {
	return &Editor{
		editor: editor,
	}
}

func (e *Editor) Launch() (string, error) {
	return LaunchEditor(e.editor)
}

func GetUserInput(cfg *internal.AppConfig) (*api.UserInput, error) {
	if cfg.Message != "" {
		return &api.UserInput{
			Message:  cfg.Message,
			Files:    cfg.Files,
			Template: cfg.Template,
		}, nil
	}

	// stdin with | or <
	var stdin io.Reader
	if cfg.Stdin || cfg.IsPiped {
		stdin = os.Stdin
	}

	input, err := userInput(cfg, stdin, util.NewClipboard(), NewEditor(cfg.Editor))
	if err != nil {
		return nil, err
	}

	//
	input.Files = cfg.Files
	input.Template = cfg.Template

	log.Debugf("\n[%s]\n%s\n%s\n%v\n\n", cfg.Me, input.Message, clipText(input.Content, clipMaxLen), input.Files)
	return input, nil
}

func userInput(
	cfg *internal.AppConfig,
	stdin io.Reader,
	clipboard ClipboardProvider,
	editor EditorProvider,
) (*api.UserInput, error) {

	msg := trimInputMessage(strings.Join(cfg.Args, " "))

	isSpecial := func() bool {
		return cfg.Stdin || cfg.Clipin || stdin != nil
	}()

	// read from command line
	if len(msg) > 0 && !isSpecial {
		return &api.UserInput{Message: msg}, nil
	}

	cat := func(msg, data string) *api.UserInput {
		m := strings.TrimSpace(msg)
		d := strings.TrimSpace(data)
		return &api.UserInput{Message: m, Content: d}
	}

	// stdin takes precedence over clipboard
	// if both are requested, stdin is used
	if stdin != nil {
		log.Promptf("Please enter your input. Ctrl+D to send, Ctrl+C to cancel...\n")
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
		return cat(msg, string(data)), nil
	}

	// clipboard
	if cfg.Clipin {
		var data string
		if cfg.ClipWait {
			// paste-append from clipboard
			var pb []string
			for {
				if err := clipboard.Clear(); err != nil {
					return nil, err
				}

				log.Promptf("Awaiting clipboard content...\n")
				v, err := clipboard.Read()
				if err != nil {
					return nil, err
				}
				log.Printf("\n%s\n\n", clipText(v, 500))
				send, err := pasteConfirm()
				// user canceled
				if err != nil {
					return nil, err
				}
				if send {
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

			log.Promptf("Awaiting clipboard content...\n")
			v, err := clipboard.Read()
			if err != nil {
				return nil, err
			}
			data = v
		}
		return cat(msg, data), nil
	}

	// no message and no special input
	// editor
	log.Debugf("Using editor: %s\n", cfg.Editor)
	content, err := editor.Launch()
	if err != nil {
		return nil, err
	}
	return cat("", content), nil
}

func LaunchEditor(editor string) (string, error) {
	tmpFile, err := os.CreateTemp("", "ai_*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return (string(content)), nil
}

// PrintInput prints the user message or intent only
func PrintInput(cfg *internal.AppConfig, input *api.UserInput) {
	if input == nil {
		return
	}

	var msg = clipText(input.Query(), clipMaxLen)
	renderInputContent(cfg.Me, msg)
}

// pasteConfirm prompts the user to append, send, or cancel the input
// and returns true if the user chooses to send the input
func pasteConfirm() (bool, error) {
	ps := "Append to input, Send, or Cancel? [A/s/c] "
	choices := []string{"append", "send", "cancel"}
	defaultChoice := "append"

	answer, err := util.Confirm(ps, choices, defaultChoice, os.Stdin)
	if err != nil {
		return false, err
	}
	if answer == "send" {
		return true, nil
	} else if answer == "append" {
		return false, nil
	}
	return false, fmt.Errorf("canceled")
}

func renderInputContent(display, content string) {
	md := util.Render(content)
	log.Infof("\n[%s]\n", display)
	log.Infoln(md)
}

// trimInputMessage trims the input message by removing leading and trailing spaces
// and also removes any trailing clipboard redirection markers.
func trimInputMessage(s string) string {
	msg := strings.TrimSpace(s)
	for {
		old := msg

		msg = strings.TrimSuffix(msg, internal.ClipoutRedirect2)
		msg = strings.TrimSuffix(msg, internal.ClipinRedirect2)
		msg = strings.TrimSuffix(msg, internal.ClipoutRedirect)
		msg = strings.TrimSuffix(msg, internal.ClipinRedirect)

		// If no markers were removed, exit the loop
		if msg == old {
			break
		}
	}
	return strings.TrimSpace(msg)
}
