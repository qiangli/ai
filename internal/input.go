package internal

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/cli/internal/log"
)

func GetUserInput(cfg *Config, args []string) (string, error) {
	// read input from stdin or clipboard
	// if the only argument is "-" or "="
	isStdin := func() bool {
		if len(args) == 2 && args[1] == "-" {
			return true
		}
		return false
	}
	isClipboard := func() bool {
		if len(args) == 2 && args[1] == "=" {
			return true
		}
		return false
	}
	isSpecial := func() bool {
		return isStdin() || isClipboard()
	}

	// read from command line
	if len(args) > 1 && !isSpecial() {
		msg := strings.Join(args[1:], " ")
		return msg, nil
	}

	// stdin with | or <
	isPiped := func() bool {
		stat, _ := os.Stdin.Stat()
		return (stat.Mode() & os.ModeCharDevice) == 0
	}

	if isPiped() || isStdin() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}

	// clipboard
	if isClipboard() {
		if err := ClearClipboard(); err != nil {
			return "", err
		}
		return ReadFromClipboard()
	}

	// editor
	editor := cfg.Editor
	log.Debugf("Using editor: %s\n", editor)

	return LaunchEditor(editor)
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
