package tool

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openai/openai-go"
)

func define(name, description string, parameters map[string]interface{}) openai.ChatCompletionToolParam {
	return openai.ChatCompletionToolParam{
		Type: openai.F(openai.ChatCompletionToolTypeFunction),
		Function: openai.F(openai.FunctionDefinitionParam{
			Name:        openai.String(name),
			Description: openai.String(description),
			Parameters:  openai.F(openai.FunctionParameters(parameters)),
		}),
	}
}

var Tools = []openai.ChatCompletionToolParam{
	define("man",
		"Retrieve the man page for a command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"command"},
		}),
	define("help",
		"Get help information for a command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "Command to get help information for",
				},
				"argument": map[string]string{
					"type":        "string",
					"description": "Flag, option, or argument to pass to the command",
				},
			},
			"required": []string{"command", "argument"},
		}),
	define("version",
		"Get version information for a command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "Command to get version information for",
				},
				"argument": map[string]string{
					"type":        "string",
					"description": "Flag, option, or argument to pass to the command",
				},
			},
			"required": []string{"command", "argument"},
		}),
	define("command",
		"Display the path or information about command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"commands": map[string]interface{}{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
					"description": "List of commands to display the path or information about",
				},
			},
			"required": []string{"commands"},
		}),
	define("which",
		"Locate a program file in the user's path",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"commands": map[string]interface{}{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
					"description": "List of command names and searches the path for each executable file that would be run had these commands actually been invoked",
				},
			},
			"required": []string{"commands"},
		}),
	define("env",
		"Print environment on user's system. Only names are returned for security reasons",
		nil,
	),
	define("pwd",
		"Return working directory name",
		nil,
	),
	define("list-commands",
		"Return a list of available commands on the system",
		nil,
	),
	define("date",
		"Display date and time",
		nil,
	),
}

func RunTool(name string, props map[string]interface{}) (string, error) {
	// Change to a temporary directory to avoid any side effects
	// This also magically fixes the following mysterious error for "man" (runMan):
	// shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
	// chdir: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
	//
	tempDir := os.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		return "", err
	}
	switch name {
	case "man":
		command := props["command"].(string)
		out, err := runMan(command)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "help":
		command := props["command"].(string)
		arg := props["argument"].(string)
		out, err := runHelp(command, arg)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "version":
		command := props["command"].(string)
		arg := props["argument"].(string)
		out, err := runVersion(command, arg)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "command":
		commands := props["commands"].([]interface{})
		items := make([]string, len(commands))
		for i, v := range commands {
			items[i] = v.(string)
		}
		out, err := runCommandV(items)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "which":
		commands := props["commands"].([]interface{})
		items := make([]string, len(commands))
		for i, v := range commands {
			items[i] = v.(string)
		}
		out, err := runWhich(items)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "env":
		out := GetEnvVarNames()
		fmt.Printf("env: %s\n", out)
	case "pwd":
		out, err := Getwd()
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "list-commands":
		out, err := ListCommandNames()
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "date":
		out, err := runDate()
		if err != nil {
			out = err.Error()
		}
		return out, nil
	default:
		return "", fmt.Errorf("unknown tool call: %s", name)
	}

	return "", nil
}

// runCommand executes a shell command with args and returns the output
func runCommand(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func runMan(command string) (string, error) {
	manCmd := exec.Command("man", command)
	var manOutput bytes.Buffer

	// Capture the output of the man command.
	manCmd.Stdout = &manOutput
	manCmd.Stderr = &manOutput

	if err := manCmd.Run(); err != nil {
		return "", fmt.Errorf("error running man command: %v\nOutput: %s", err, manOutput.String())
	}

	// Process with 'col' to remove formatting
	colCmd := exec.Command("col", "-b")
	var colOutput bytes.Buffer

	colCmd.Stdin = bytes.NewReader(manOutput.Bytes())
	colCmd.Stdout = &colOutput
	colCmd.Stderr = &colOutput

	// Try running 'col', if it fails, return the man output instead.
	if err := colCmd.Run(); err != nil {
		return manOutput.String(), nil
	}

	return colOutput.String(), nil
}

// isSafeArg checks if arg is a command line option by the common convention
// namely prefixed with "-" (including "--") or is one of the allowed arguments
func isSafeArg(arg string, allowed []string) bool {
	if strings.HasPrefix(arg, "-") {
		return true
	}

	// Check against the list of allowed arguments
	for _, v := range allowed {
		if arg == v {
			return true
		}
	}

	return false
}

// runHelp retrieves the help output for a command
func runHelp(command string, arg string) (string, error) {
	const tpl = `
Invalid argument '%s' detected for command '%s'.
Accepted format: any flag starting with '-' or '--' or one of the following: %v.
Consider consulting the command's man page or using the help option to find the correct argument.
	`
	allowed := []string{"--help", "help"}
	if !isSafeArg(arg, allowed) {
		return "", fmt.Errorf(tpl, arg, command, allowed)
	}
	out, err := exec.Command(command, arg).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// runVersion retrieves the version output for a command
func runVersion(command string, arg string) (string, error) {
	const tpl = `
Invalid argument '%s' detected for command '%s'.
Accepted format: any flag starting with '-' or '--' or one of the following: %v.
Consider consulting the command's man page or using the help option to find the correct argument.
	`
	allowed := []string{"--version", "version"}
	if !isSafeArg(arg, allowed) {
		return "", fmt.Errorf(tpl, arg, command, allowed)
	}
	out, err := exec.Command(command, arg).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func GetEnvVarNames() string {
	names := []string{}
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			names = append(names, pair[0])
		}
	}
	sort.Strings(names)
	return strings.Join(names, "\n")
}

func Getwd() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return pwd, nil
}

// runCommandV executes the shell "command -v" with a list of commands and returns the output
func runCommandV(commands []string) (string, error) {
	args := append([]string{"-v"}, commands...)
	return runCommand("command", args...)
}

func runWhich(commands []string) (string, error) {
	return runCommand("which", commands...)
}

func runDate() (string, error) {
	return runCommand("date")
}

func ListCommandNames() (string, error) {
	list, err := ListCommands(true)
	if err != nil {
		return "", err
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), nil
}

// ListCommands returns the full path of the first valid executable command encountered in the PATH
func ListCommands(nameOnly bool) ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, errors.New("PATH environment variable is not set")
	}

	uniqueCommands := make(map[string]string) // command name -> full path
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, pathDir := range paths {
		files, err := os.ReadDir(pathDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			commandName := file.Name()
			fullPath := filepath.Join(pathDir, commandName)

			// Check if the file is executable and the command hasn't been registered yet
			if !file.IsDir() && IsExecutable(fullPath) {
				if _, exists := uniqueCommands[commandName]; !exists {
					uniqueCommands[commandName] = fullPath
				}
			}
		}
	}

	commands := make([]string, 0, len(uniqueCommands))
	for name, fullPath := range uniqueCommands {
		if nameOnly {
			commands = append(commands, name)
			continue
		}
		commands = append(commands, fullPath)
	}

	return commands, nil
}

func IsExecutable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode.IsRegular() && mode&0111 != 0
}
