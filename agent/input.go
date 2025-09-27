package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal"
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
func GetUserInput(ctx context.Context, cfg *api.AppConfig) (*api.UserInput, error) {
	return getUserInput(ctx, cfg, nil, nil, nil)
}

func getUserInput(ctx context.Context, cfg *api.AppConfig, stdin io.Reader, clipper api.ClipboardProvider, editor api.EditorProvider) (*api.UserInput, error) {
	// --message flag - ignore the rest (mainly intended for testing)
	if cfg.Message != "" {
		input := &api.UserInput{
			Message:  cfg.Message,
			Files:    cfg.Files,
			Template: cfg.Template,
		}
		return input, nil
	}

	// collecting message content from various sources
	if clipper == nil {
		clipper = util.NewClipboard()
	}
	if editor == nil {
		editor = NewEditor(cfg.Editor)
	}

	input, err := userInput(ctx, cfg, stdin, clipper, editor)
	if err != nil {
		return nil, err
	}

	// attachments
	input.Files = cfg.Files
	input.Template = cfg.Template

	// special inputs

	// // take screenshot - append to file list
	// if cfg.Screenshot {
	// 	if img, err := takeScreenshot(cfg); err != nil {
	// 		return nil, err
	// 	} else {
	// 		input.Files = append(input.Files, img)
	// 	}
	// }

	// // get voice input - append to message
	// if cfg.Voice {
	// 	if txt, err := voiceInput(cfg); err != nil {
	// 		return nil, err
	// 	} else {
	// 		input.Message = input.Message + " " + txt
	// 	}
	// }

	log.GetLogger(ctx).Debugf("\n[%s]\n%s\n%s\n%v\n", cfg.Me, input.Message, clipText(input.Content, clipMaxLen), input.Files)
	return input, nil
}

func userInput(
	ctx context.Context,
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
		log.GetLogger(ctx).Promptf("Please enter your input. Ctrl+D to send, Ctrl+C to cancel...\n")
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

	// send all collected inputs to the editor and start the editor
	content = cat(msg, content, "\n")

	if cfg.Editor != "" {
		log.GetLogger(ctx).Debugf("Using editor: %s\n", cfg.Editor)
		data, err := editor.Launch(content)
		if err != nil {
			return nil, err
		}
		return &api.UserInput{Content: data}, nil
	}

	data, canceled, err := SimpleEditor(cfg.Me.Display, content)
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

// PrintInput prints the user input
func PrintInput(ctx context.Context, cfg *api.AppConfig, input *api.UserInput) {
	if input == nil {
		return
	}
	log.GetLogger(ctx).Debugf("UserInput:\n%s\n", input.String())

	// query and files for info only
	var msg = clipText(input.Query(), clipMaxLen)
	for _, v := range input.Files {
		msg += fmt.Sprintf("\n+ %s", v)
	}
	renderInputContent(ctx, cfg.Me, msg)

	// attachments
	for _, v := range input.Files {
		ext := filepath.Ext(v)
		var emoji string
		// TODO more extensions
		switch ext {
		case "txt", "yaml", "yml", "md":
			emoji = "📄"
		case "png", "jpg", "jpeg", "gif", "webp":
			emoji = "🖼️"
		default:
			emoji = "💾"
		}
		log.GetLogger(ctx).Infof("%s attachment: %s\n", emoji, v)
	}
	// for _, v := range input.Messages {
	// 	var emoji string
	// 	ps := strings.SplitN(v.ContentType, "/", 2)
	// 	switch ps[0] {
	// 	case "text":
	// 		emoji = "📄"
	// 	case "image":
	// 		emoji = "🖼️"
	// 	default:
	// 		emoji = "💾"
	// 	}
	// 	var content string
	// 	if v.ContentType == "" || strings.HasPrefix(v.ContentType, "text/") {
	// 		content = clipText(v.Content, 100)
	// 	} else if strings.HasPrefix(v.ContentType, "image/") && strings.HasPrefix(v.Content, "data:") {
	// 		content = clipText(v.Content, 100)
	// 	} else {
	// 		content = "[binary]"
	// 	}
	// 	log.GetLogger(ctx).Infof("%s attachment type: %s Len: %v content: %s\n", emoji, v.ContentType, len(v.Content), content)
	// }
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
	if answer == "send" {
		return true, nil
	} else if answer == "append" {
		return false, nil
	}
	return false, fmt.Errorf("canceled")
}

func renderInputContent(ctx context.Context, me *api.User, content string) {
	md := util.Render(content)
	log.GetLogger(ctx).Infof("\n[%s]\n%s\n", me.Display, md)
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
