package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/cli/internal/log"
)

func GetUserInput(cfg *Config, args []string) (string, error) {
	// stdin with | or <
	isPiped := func() bool {
		stat, _ := os.Stdin.Stat()
		return (stat.Mode() & os.ModeCharDevice) == 0
	}()

	var lastChar string
	var msg string

	// /bin message...
	if len(args) > 1 {
		msg = strings.TrimSpace(strings.Join(args[1:], " "))
		// read input from stdin or clipboard
		// if the last char on the command line is "-" or "="
		if !isPiped && len(msg) > 0 {
			lastChar = string(msg[len(msg)-1])
			if lastChar == "-" || lastChar == "=" {
				msg = strings.TrimSpace(msg[:len(msg)-1])
			} else {
				lastChar = ""
			}
		}
	}

	isStdin := func() bool {
		return lastChar == "-"
	}()
	isClipboard := func() bool {
		return lastChar == "="
	}()

	isSpecial := func() bool {
		return isStdin || isClipboard || isPiped
	}()

	// read from command line
	if len(msg) > 0 && !isSpecial {
		return msg, nil
	}

	// keep this format!
	const msgDataTpl = `###
%s

###
%s
`
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
	if isPiped || isStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", err
		}
		return cat(msg, string(data)), nil
	}

	// clipboard
	if isClipboard {
		if err := ClearClipboard(); err != nil {
			return "", err
		}
		data, err := ReadFromClipboard()
		if err != nil {
			return "", err
		}
		return cat(msg, data), nil
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
