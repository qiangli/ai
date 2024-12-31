package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/cli/internal/log"
)

const StdinInputRedirect = "-"

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

func GetUserInput(cfg *Config) (string, error) {
	// stdin with | or <
	isPiped := func() bool {
		stat, _ := os.Stdin.Stat()
		return (stat.Mode() & os.ModeCharDevice) == 0
	}()

	var stdin io.Reader
	if cfg.Stdin || isPiped {
		stdin = os.Stdin
	}

	return userInput(cfg, stdin, NewClipboard(), NewEditor(cfg.Editor))
}

func userInput(
	cfg *Config,
	stdin io.Reader,
	clipboard ClipboardProvider,
	editor EditorProvider,
) (string, error) {

	msg := strings.TrimSpace(strings.Join(cfg.Args, " "))

	isSpecial := func() bool {
		return cfg.Stdin || cfg.Clipin || stdin != nil
	}()

	// read from command line
	if len(msg) > 0 && !isSpecial {
		return msg, nil
	}

	const msgDataTpl = "###\n%s\n###\n%s\n"
	cat := func(msg, data string) string {
		m := strings.TrimSpace(msg)
		d := strings.TrimSpace(data)
		switch {
		case m == "" && d == "":
			return ""
		case m == "":
			return d
		case d == "":
			return m
		default:
			return fmt.Sprintf(msgDataTpl, m, d)
		}
	}

	// stdin takes precedence over clipboard
	// if both are requested, stdin is used
	if stdin != nil {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}
		return cat(msg, string(data)), nil
	}

	// clipboard
	if cfg.Clipin {
		if err := clipboard.Clear(); err != nil {
			return "", err
		}
		data, err := clipboard.Read()
		if err != nil {
			return "", err
		}
		return cat(msg, data), nil
	}

	// no message and no special input
	// editor
	log.Debugf("Using editor: %s\n", cfg.Editor)
	return editor.Launch()
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
