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

const StdinRedirect = "-"

type ClipboardProvider = util.ClipboardProvider

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

	msg := strings.TrimSpace(strings.Join(cfg.Args, " "))

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
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
		return cat(msg, string(data)), nil
	}

	// clipboard
	if cfg.Clipin {
		if err := clipboard.Clear(); err != nil {
			return nil, err
		}

		var data string
		if cfg.ClipWait {
			// TODO read-append clipboard content
			v, err := clipboard.Read()
			if err != nil {
				return nil, err
			}
			log.Printf("%s\n", clipText(v, 100))
			if err := Confirm(); err != nil {
				return nil, err
			}
			data = v
		} else {
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

	if cfg.Format == "markdown" {
		renderContent(cfg.Me, msg)
	} else {
		showContent(cfg.Me, msg)
	}
}

func Confirm() error {
	ps := "Confirm? [Y/n] "
	choices := []string{"yes", "no"}
	defaultChoice := "yes"

	answer, err := util.Confirm(ps, choices, defaultChoice, os.Stdin)
	if err != nil {
		return err
	}
	if answer == "yes" {
		return nil
	}
	return fmt.Errorf("canceled")
}
