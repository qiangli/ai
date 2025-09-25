package swarm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/log"
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

func readClipboard(ctx context.Context) (string, error) {
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

func readClipboardWait(ctx context.Context) (string, error) {
	c := util.NewClipboard()
	if err := c.Clear(); err != nil {
		return "", err
	}
	data, err := c.Read()
	if err != nil {
		return "", err
	}

	if err := confirmReadClipboard(ctx); err != nil {
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

func confirmReadClipboard(ctx context.Context) error {
	ps := "Confirm? [Y/n] "
	choices := []string{"yes", "no"}
	defaultChoice := "yes"

	answer, err := util.Confirm(ctx, ps, choices, defaultChoice, os.Stdin)
	if err != nil {
		return err
	}
	if answer == "yes" {
		return nil
	}
	return fmt.Errorf("clipboard read canceled")
}

func getUserTextInput(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt is empty")
	}
	ps := fmt.Sprintf("\n%s:\n\n[Press Ctrl+D to send or Ctrl+C to cancel...]\n", prompt)
	log.GetLogger(ctx).Promptf(ps)
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func getUserChoiceInput(ctx context.Context, prompt string, choices []string, defaultChoice string) (string, error) {
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
	return util.Confirm(ctx, ps, choices, defaultChoice, os.Stdin)
}

// if required properties is not missing and is an array of strings
// check if the required properties are present
func isRequired(key string, props map[string]any) bool {
	val, ok := props["required"]
	if !ok {
		return false
	}
	items, ok := val.([]string)
	if !ok {
		return false
	}
	for _, v := range items {
		if v == key {
			return true
		}
	}
	return false
}

func GetStrProp(key string, props map[string]any) (string, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return "", fmt.Errorf("missing property: %s", key)
		}
		return "", nil
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("property '%s' must be a string", key)
	}
	return str, nil
}

func GetIntProp(key string, props map[string]any) (int, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return 0, fmt.Errorf("missing property: %s", key)
		}
		return 0, nil
	}
	switch v := val.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		s := fmt.Sprintf("%v", val)
		return strconv.Atoi(s)
	}
}

func GetArrayProp(key string, props map[string]any) ([]string, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return nil, fmt.Errorf("missing property: %s", key)
		}
		return []string{}, nil
	}
	items, ok := val.([]any)
	if ok {
		strs := make([]string, len(items))
		for i, v := range items {
			str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be an array of strings", key)
			}
			strs[i] = str
		}
		return strs, nil
	}

	strs, ok := val.([]string)
	if !ok {
		if isRequired(key, props) {
			return nil, fmt.Errorf("%s must be an array of strings", key)
		}
		return []string{}, nil
	}
	return strs, nil
}
