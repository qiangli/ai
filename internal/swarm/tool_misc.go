package swarm

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

type Descriptor struct {
	Name        string
	Description string
	Parameters  map[string]any
}

const (
	ReadStdinToolName            = "read_stdin"
	ReadClipboardToolName        = "read_clipboard"
	ReadClipboardWaitToolName    = "read_clipboard_wait"
	WriteStdoutToolName          = "write_stdout"
	WriteClipboardToolName       = "write_clipboard"
	WriteClipboardAppendToolName = "write_clipboard_append"
	GetUserTextInputToolName     = "get_user_text_input"
	GetUserChoiceInputToolName   = "get_user_choice_input"
)

// Miscellaneous system tools
var miscDescriptors = map[string]*Descriptor{
	ReadStdinToolName: {
		Name:        ReadStdinToolName,
		Description: "Read input from stdin",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	ReadClipboardToolName: {
		Name:        ReadClipboardToolName,
		Description: "Read input from the clipboard without waiting for user's confirmation",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	ReadClipboardWaitToolName: {
		Name:        ReadClipboardWaitToolName,
		Description: "Read input from the clipboard and wait for user's confirmation",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	},
	WriteStdoutToolName: {
		Name:        WriteStdoutToolName,
		Description: "Write output to stdout/console",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the console",
				},
			},
			"required": []string{"content"},
		},
	},
	WriteClipboardToolName: {
		Name:        WriteClipboardToolName,
		Description: "Copy output to the clipboard and overwrite its existing content",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the clipboard",
				},
			},
			"required": []string{"content"},
		},
	},
	WriteClipboardAppendToolName: {
		Name:        WriteClipboardAppendToolName,
		Description: "Append content to the clipboard",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{
					"type":        "string",
					"description": "Content to append to the clipboard",
				},
			},
			"required": []string{"content"},
		},
	},
	GetUserTextInputToolName: {
		Name:        GetUserTextInputToolName,
		Description: "Get additional text input for clarification",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "Prompt for input",
					"example":     "Please provide more details",
					"maxLength":   100,
					"minLength":   1,
				},
			},
			"required": []string{"prompt"},
		},
	},
	GetUserChoiceInputToolName: {
		Name:        GetUserChoiceInputToolName,
		Description: "Get confirmation or approval",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{
					"type":        "string",
					"description": "Prompt for input",
					"example":     "Please confirm the action",
					"maxLength":   100,
					"minLength":   1,
				},
				"choices": map[string]any{
					"type":        "array",
					"description": "List of options to choose from",
					"items": map[string]any{
						"type": "string",
					},
					"example": []string{"yes", "no"},
				},
				"default": map[string]any{
					"type":        "string",
					"description": "Default choice if none is selected",
					"example":     "yes",
				},
			},
			"required": []string{"prompt", "choices", "default"},
		},
	},
}

func ListMiscTools() ([]*ToolFunc, error) {
	var tools []*ToolFunc
	for k, v := range miscDescriptors {
		tools = append(tools, &ToolFunc{
			Name:        k,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

	sortTools(tools)

	return tools, nil
}

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
