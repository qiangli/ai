package swarm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func readStdin() (string, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeStdout(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	_, err := fmt.Fprintln(os.Stdout, content)
	if err != nil {
		return "", err
	}
	return content, nil
}

func readClipboard() (string, error) {
	c := util.NewClipboard()
	if err := c.Clear(); err != nil {
		return "", err
	}
	data, err := c.Read()
	if err != nil {
		return "", err
	}
	return data, nil
}

func readClipboardWait() (string, error) {
	c := util.NewClipboard()
	if err := c.Clear(); err != nil {
		return "", err
	}
	data, err := c.Read()
	if err != nil {
		return "", err
	}

	if err := confirmReadClipboard(); err != nil {
		return "", err
	}

	return data, nil
}

func writeClipboard(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	c := util.NewClipboard()
	if err := c.Write(content); err != nil {
		return "", err
	}
	return content, nil
}

func writeClipboardAppend(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	c := util.NewClipboard()
	if err := c.Append(content); err != nil {
		return "", err
	}
	return content, nil
}

func confirmReadClipboard() error {
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
	return fmt.Errorf("clipboard read canceled")
}

func getUserTextInput(prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is empty")
	}
	ps := fmt.Sprintf("\n%s:\n\n[Press Ctrl+D to send or Ctrl+C to cancel...]\n", prompt)
	log.Prompt(ps)
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getUserChoiceInput(prompt string, choices []string, defaultChoice string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is empty")
	}
	if len(choices) == 0 {
		return "", fmt.Errorf("choices are empty")
	}
	if defaultChoice == "" {
		return "", fmt.Errorf("default choice is empty")
	}
	if len(choices) < 2 {
		return "", fmt.Errorf("choices must have at least two options")
	}
	ps := fmt.Sprintf("\n%s:\n\nPress enter to respond with %q [%s] ", prompt, strings.ToLower(defaultChoice), strings.Join(choices, "/"))
	return util.Confirm(ps, choices, defaultChoice, os.Stdin)
}
