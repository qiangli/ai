package tool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/util"
)

type Config struct {
	DBConfig *db.DBConfig
}

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

var SystemTools = []openai.ChatCompletionToolParam{
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
	define("uname",
		"Display information about the system",
		nil,
	),
	define("ls",
		"list directory contents and requested info for each file operand, including directory contents. No operands mean listing the current directory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"args": map[string]interface{}{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
					"description": "files or directories and flags",
				},
			},
			"required": []string{"args"},
		}),
	define("cd",
		"change the current directory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"dir": map[string]string{
					"type":        "string",
					"description": "directory to change to",
				},
			},
			"required": []string{"dir"},
		}),
	define("mkdir",
		"make a directory. parent directories are created as needed",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"dir": map[string]string{
					"type":        "string",
					"description": "directory to make",
				},
			},
			"required": []string{"dir"},
		}),
	define("git-status",
		"get the status of the git repository",
		nil),
}

func RunTool(cfg *Config, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	if strings.HasPrefix(name, "ai_") {
		return runAIHelpTool(cfg, ctx, name, props)
	}
	if strings.HasPrefix(name, "db_") {
		return runDbTool(cfg, ctx, name, props)
	}
	return runCommandTool(cfg, ctx, name, props)
}

func runCommandTool(_ *Config, _ context.Context, name string, props map[string]interface{}) (string, error) {
	// Change to a temporary directory to avoid any side effects
	// This also magically fixes the following mysterious error for "man" (runMan):
	// shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
	// chdir: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
	//
	tempDir := os.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		return "", err
	}

	getStr := func(key string) (string, error) {
		val, ok := props[key]
		if !ok {
			return "", fmt.Errorf("missing property: %s", key)
		}
		str, ok := val.(string)
		if !ok {
			return "", fmt.Errorf("property '%s' must be a string", key)
		}
		return str, nil
	}

	getArray := func(key string) ([]string, error) {
		val, ok := props[key]
		if !ok {
			return nil, fmt.Errorf("missing property: %s", key)
		}
		items, ok := val.([]interface{})
		if !ok {
			return nil, fmt.Errorf("property '%s' must be an array of strings", key)
		}
		array := make([]string, len(items))
		for i, v := range items {
			array[i] = v.(string)
		}
		return array, nil
	}

	switch name {
	case "man":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		out, err := runMan(command)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "help":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		arg, err := getStr("argument")
		if err != nil {
			return "", err
		}
		out, err := runHelp(command, arg)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "version":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		arg, err := getStr("argument")
		if err != nil {
			return "", err
		}
		out, err := runVersion(command, arg)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "command":
		commands, err := getArray("commands")
		if err != nil {
			return "", err
		}
		out, err := runCommandV(commands)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "which":
		commands, err := getArray("commands")
		if err != nil {
			return "", err
		}
		out, err := runCommand("which", commands...)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "env":
		out := util.GetEnvVarNames()
		return out, nil
	case "pwd":
		out, err := util.Getwd()
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
		out, err := runCommand("date")
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "uname":
		as, arch := util.Uname()
		return fmt.Sprintf("OS: %s\nArch: %s", as, arch), nil
	case "ls":
		args, err := getArray("args")
		if err != nil {
			return "", err
		}
		out, err := runCommand("ls", args...)
		if err != nil {
			out = err.Error()
		}
		return out, nil
	case "cd":
		dir, err := getStr("dir")
		if err != nil {
			return "", err
		}
		if err := os.Chdir(dir); err != nil {
			return "", err
		}
		return "", nil
	case "mkdir":
		dir, err := getStr("dir")
		if err != nil {
			return "", err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}
		return "", nil
	case "git-status":
		out, err := runCommand("git", "status")
		if err != nil {
			out = err.Error()
		}
		return out, nil
	}

	return "", fmt.Errorf("unknown tool: %s", name)
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

// runCommandV executes the shell "command -v" with a list of commands and returns the output
func runCommandV(commands []string) (string, error) {
	args := append([]string{"-v"}, commands...)
	return runCommand("command", args...)
}

func ListCommandNames() (string, error) {
	list, err := util.ListCommands(true)
	if err != nil {
		return "", err
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), nil
}
