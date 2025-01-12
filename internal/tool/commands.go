package tool

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/tool/vos"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os
var _util = _os

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
	define("list-commands",
		"return a list of available commands on the system",
		nil,
	),
	define("man",
		"find and display online manual documentation page for a command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "The command to get the manual page for",
				},
			},
			"required": []string{"command"},
		},
	),
	define("help",
		"display helpful information about a command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "The command to get help information for",
				},
				"argument": map[string]string{
					"type":        "string",
					"description": "Flag, option, or argument to pass to the command. Commonly '--help', '-h', or 'help'",
				},
			},
			"required": []string{"command", "argument"},
		},
	),
	define("version",
		"get version information for a command",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "Command to get version information for",
				},
				"argument": map[string]string{
					"type":        "string",
					"description": "Flag, option, or argument to pass to the command. Commonly '--version', '-v', or 'version'",
				},
			},
			"required": []string{"command", "argument"},
		},
	),
	define("command",
		"display the path or information about commands",
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
		},
	),
	define("which",
		"locate a program file in the user's path",
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
		},
	),
	define("exec",
		"execute a command. Restrictions apply",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]string{
					"type":        "string",
					"description": "The command to execute",
				},
				"args": map[string]interface{}{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
					"description": "optional arguments to pass to the command",
				},
			},
			"required": []string{"command"},
		},
	),
	define("env",
		"print environment on user's system. Only names are returned for security reasons",
		nil,
	),
	define("pwd",
		"return working directory name",
		nil,
	),
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
		},
	),
	define("ls",
		"list directory contents as well as any requested, associated information",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"args": map[string]interface{}{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
					"description": "files, directories and flags",
				},
			},
		},
	),
	define("mkdir",
		"make a directory. parent directories are created as needed",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"dir": map[string]string{
					"type":        "string",
					"description": "The directory to make",
				},
			},
			"required": []string{"dir"},
		},
	),
	define("uname",
		"display information about the current system's operating system and architecture",
		nil,
	),
	define("test",
		"condition evaluation utility. if it evaluates to true, returns true; otherwise it returns false.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"expression": map[string]interface{}{
					"type":        "string",
					"description": "The expression to evaluate",
				},
			},
			"required": []string{"expression"},
		},
	),
}

func RunTool(cfg *Config, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	var out string
	var err error

	switch {
	case strings.HasPrefix(name, "ai_"):
		out, err = runAIHelpTool(cfg, ctx, name, props)
	case strings.HasPrefix(name, "db_"):
		out, err = runDbTool(cfg, ctx, name, props)
	default:
		out, err = runCommandTool(cfg, ctx, name, props)
	}

	if err != nil {
		return err.Error(), nil
	}
	return out, nil
}

func runCommandTool(_ *Config, _ context.Context, name string, props map[string]interface{}) (string, error) {
	// if required properties is not missing and is an array of strings
	// check if the required properties are present
	required := func(key string) bool {
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

	getStr := func(key string) (string, error) {
		val, ok := props[key]
		if !ok {
			if required(key) {
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

	getArray := func(key string) ([]string, error) {
		val, ok := props[key]
		if !ok {
			if required(key) {
				return nil, fmt.Errorf("name: %s missing property: %s", name, key)
			}
			return []string{}, nil
		}
		items, ok := val.([]string)
		if !ok {
			if required(key) {
				return nil, fmt.Errorf("name: %s property '%s' must be an array of strings", name, key)
			}
			return []string{}, nil
		}
		return items, nil
	}

	// shell commands sensitive to the current directory or relatively "safe" to run
	switch name {
	case "exec":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		args, err := getArray("args")
		if err != nil {
			return "", err
		}
		return safeRunCommand(command, args...)
	case "pwd":
		return _os.Getwd()
	case "cd":
		dir, err := getStr("dir")
		if err != nil {
			return "", err
		}
		return "", _os.Chdir(dir)
	case "ls":
		args, err := getArray("args")
		if err != nil {
			return "", err
		}
		return runCommand("ls", args...)
	case "mkdir":
		dir, err := getStr("dir")
		if err != nil {
			return "", err
		}
		return "", _os.MkdirAll(dir, 0755)
	case "test":
		expr, err := getStr("expression")
		if err != nil {
			return "", err
		}
		_, err = runCommand("test", expr)
		if err != nil {
			return "false", nil
		}
		return "true", nil
	}

	// Change to a temporary directory to avoid any side effects
	// This also magically fixes the following mysterious error for "man" (runMan):
	// shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
	// chdir: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
	//
	curDir, err := _os.Getwd()
	if err != nil {
		return "", err
	}
	defer func() {
		err := _os.Chdir(curDir)
		if err != nil {
			log.Errorf("runCommandTool error changing back to original directory:", err)
		}
	}()
	tempDir := _os.TempDir()
	if err := _os.Chdir(tempDir); err != nil {
		return "", err
	}

	switch name {
	case "list-commands":
		return ListCommandNames()
	case "man":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		return runMan(command)
	case "help":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		arg, err := getStr("argument")
		if err != nil {
			return "", err
		}
		return runHelp(command, arg)
	case "version":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		arg, err := getStr("argument")
		if err != nil {
			return "", err
		}
		return runVersion(command, arg)
	case "command":
		commands, err := getArray("commands")
		if err != nil {
			return "", err
		}
		return runCommandV(commands)
	case "which":
		commands, err := getArray("commands")
		if err != nil {
			return "", err
		}
		return runCommand("which", commands...)
	case "env":
		return _util.Env(), nil
	case "uname":
		as, arch := _util.Uname()
		return fmt.Sprintf("OS: %s\nArch: %s", as, arch), nil
	}

	return "", fmt.Errorf("unknown tool: %s", name)
}

// runCommand executes a shell command with args and returns the output
func runCommand(cmd string, args ...string) (string, error) {
	out, err := _exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func runMan(command string) (string, error) {
	manCmd := _exec.Command("man", command)
	var manOutput bytes.Buffer

	// Capture the output of the man command.
	manCmd.Stdout = &manOutput
	manCmd.Stderr = &manOutput

	if err := manCmd.Run(); err != nil {
		return "", fmt.Errorf("error running man command: %v\nOutput: %s", err, manOutput.String())
	}

	// Process with 'col' to remove formatting
	colCmd := _exec.Command("col", "-b")
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
Consult the command's man page to find the valid argument.
	`
	// TODO -h may not always be safe?
	allowed := []string{"--help", "-h", "help"}
	if !isSafeArg(arg, allowed) {
		return "", fmt.Errorf(tpl, arg, command, allowed)
	}
	out, err := _exec.Command(command, arg).CombinedOutput()
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
Consult the command's man page or use the help option to find the valid argument.
	`
	// TODO -v or version may not always be safe for certain commands?
	allowed := []string{"--version", "-v", "version"}
	if !isSafeArg(arg, allowed) {
		return "", fmt.Errorf(tpl, arg, command, allowed)
	}
	out, err := _exec.Command(command, arg).CombinedOutput()
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
	list, err := _util.ListCommands(true)
	if err != nil {
		return "", err
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), nil
}

func safeRunCommand(command string, args ...string) (string, error) {
	allowed := getAllowedCommands()
	for _, v := range allowed {
		if command == v {
			return runCommand(command, args...)

		}
	}
	return "", fmt.Errorf("%s: Not permitted", command)
}

func getAllowedCommands() []string {
	return execWhitelist
}
