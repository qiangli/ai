package agent

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/cb"
	"github.com/qiangli/ai/internal/log"
)

const clipMaxLen = 500

const StdinInputRedirect = "-"

type ClipboardProvider = cb.ClipboardProvider

// clipboard redirection
const (
	// read from clipboard
	ClipboardInputRedirect = "="

	// write to clipboard
	ClipboardOutputRedirect = "=+"
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

func GetUserInput(cfg *internal.AppConfig) (*UserInput, error) {
	// stdin with | or <
	isPiped := func() bool {
		stat, _ := os.Stdin.Stat()
		return (stat.Mode() & os.ModeCharDevice) == 0
	}()

	var stdin io.Reader
	if cfg.Stdin || isPiped {
		stdin = os.Stdin
	}

	input, err := userInput(cfg, stdin, cb.NewClipboard(), NewEditor(cfg.Editor))
	if err != nil {
		return nil, err
	}

	input.Files = cfg.Files

	log.Infof("\n[%s]\n%s\n%s\n%v\n\n", cfg.Me, input.Message, clipText(input.Content, clipMaxLen), input.Files)
	return input, nil
}

func userInput(
	cfg *internal.AppConfig,
	stdin io.Reader,
	clipboard ClipboardProvider,
	editor EditorProvider,
) (*UserInput, error) {

	msg := strings.TrimSpace(strings.Join(cfg.Args, " "))

	isSpecial := func() bool {
		return cfg.Stdin || cfg.Clipin || stdin != nil
	}()

	// read from command line
	if len(msg) > 0 && !isSpecial {
		return &UserInput{Message: msg}, nil
	}

	cat := func(msg, data string) *UserInput {
		m := strings.TrimSpace(msg)
		d := strings.TrimSpace(data)
		return &UserInput{Message: m, Content: d}
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
		data, err := clipboard.Read()
		if err != nil {
			return nil, err
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
