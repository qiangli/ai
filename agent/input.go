package agent

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
)

const clipMaxLen = 500

// type ClipboardProvider = util.ClipboardProvider

// type EditorProvider interface {
// 	Launch() (string, error)
// }

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
// otherwise, it determines the input source (stdin, editor, clipboard)
// and collects input accordingly. It also
// attaches any provided files or template info if provided.
func GetUserInput(cfg *api.AppConfig) (*api.UserInput, error) {
	return getUserInput(cfg, nil, nil, nil)
}

func getUserInput(cfg *api.AppConfig, stdin io.Reader, clipper api.ClipboardProvider, editor api.EditorProvider) (*api.UserInput, error) {
	if cfg.Message != "" {
		input := &api.UserInput{
			Message: cfg.Message,
			// Content:  cfg.Content,
			Files:    cfg.Files,
			Template: cfg.Template,
		}
		return input, nil
	}

	if clipper == nil {
		clipper = util.NewClipboard()
	}
	if editor == nil {
		editor = NewEditor(cfg.Editor)
	}

	input, err := userInput(cfg, stdin, clipper, editor)
	if err != nil {
		return nil, err
	}
	//
	input.Files = cfg.Files
	input.Template = cfg.Template

	// special inputs
	// take screenshot
	if cfg.Screenshot {
		if img, err := takeScreenshot(cfg); err != nil {
			return nil, err
		} else {
			input.Files = append(input.Files, img)
		}
	}

	// get voice input
	if cfg.Voice {
		if txt, err := voiceInput(cfg); err != nil {
			return nil, err
		} else {
			input.Message = input.Message + " " + txt
		}
	}

	log.Debugf("\n[%s]\n%s\n%s\n%v\n\n", cfg.Me, input.Message, clipText(input.Content, clipMaxLen), input.Files)
	return input, nil
}

func userInput(
	cfg *api.AppConfig,
	stdin io.Reader,
	clipboard api.ClipboardProvider,
	editor api.EditorProvider,
) (*api.UserInput, error) {
	var msg = trimInputMessage(strings.Join(cfg.Args, " "))

	// stdin
	var stdinData string
	if cfg.IsStdin() {
		if stdin == nil {
			stdin = os.Stdin
		}
		log.Promptf("Please enter your input. Ctrl+D to send, Ctrl+C to cancel...\n")
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}

		stdinData = strings.TrimSpace(string(data))
	}

	// clipboard
	var clipinData string
	if cfg.IsClipin() {
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
				log.Infof("\n%s\n\n", clipText(v, 500))
				send, err := pasteConfirm()
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

			log.Promptf("Awaiting clipboard content...\n")
			v, err := clipboard.Read()
			if err != nil {
				return nil, err
			}
			data = v
		}

		clipinData = strings.TrimSpace(data)
	}

	cat := func(a, b, sep string) string {
		if a != "" && b == "" {
			return a
		} else if a == "" && b != "" {
			return b
		} else if stdinData != "" && clipinData != "" {
			return a + sep + b
		}
		return ""
	}

	var content = cat(stdinData, clipinData, "\n")

	//
	if !cfg.Editing {
		return &api.UserInput{Message: msg, Content: content}, nil
	}

	// send all inputs to the editor
	content = cat(msg, content, "\n")

	if cfg.Editor != "" {
		log.Debugf("Using editor: %s\n", cfg.Editor)
		data, err := editor.Launch(content)
		if err != nil {
			return nil, err
		}
		return &api.UserInput{Content: data}, nil
	}

	data, canceled, err := SimpleEditor(cfg.Me, content)
	if err != nil {
		return nil, err
	}
	if canceled {
		return &api.UserInput{}, nil
	}

	return &api.UserInput{Content: data}, nil
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

// split s into length of around 80 char delimited by space
func wrapByLength(s string, limit int) []string {
	words := strings.Fields(s)
	var lines []string
	var buf strings.Builder

	for _, word := range words {
		// If adding this word would exceed the limit, start a new line
		if buf.Len() > 0 && buf.Len()+len(word)+1 > limit {
			lines = append(lines, buf.String())
			buf.Reset()
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(word)
	}
	// Add the last line if any
	if buf.Len() > 0 {
		lines = append(lines, buf.String())
	}
	return lines
}

// PrintInput prints the user message or intent only
func PrintInput(cfg *api.AppConfig, input *api.UserInput) {
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
