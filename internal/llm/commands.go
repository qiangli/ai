package llm

// import (
// 	"bytes"
// 	"context"
// 	"fmt"
// 	"sort"
// 	"strings"

// 	"github.com/openai/openai-go"
// 	"github.com/qiangli/ai/internal"
// 	"github.com/qiangli/ai/internal/llm/vos"
// 	"github.com/qiangli/ai/internal/log"
// )

// var _os vos.System = &vos.VirtualSystem{}
// var _exec = _os
// var _util = _os

// func define(name, description string, parameters map[string]interface{}) openai.ChatCompletionToolParam {
// 	return openai.ChatCompletionToolParam{
// 		Type: openai.F(openai.ChatCompletionToolTypeFunction),
// 		Function: openai.F(openai.FunctionDefinitionParam{
// 			Name:        openai.String(name),
// 			Description: openai.String(description),
// 			Parameters:  openai.F(openai.FunctionParameters(parameters)),
// 		}),
// 	}
// }

// var systemTools = []openai.ChatCompletionToolParam{
// 	define("list-commands",
// 		"return a list of available commands on the system",
// 		nil,
// 	),
// 	define("man",
// 		"find and display online manual documentation page for a command",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"command": map[string]string{
// 					"type":        "string",
// 					"description": "The command to get the manual page for",
// 				},
// 			},
// 			"required": []string{"command"},
// 		},
// 	),
// 	define("command",
// 		"display the path or information about commands",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"commands": map[string]interface{}{
// 					"type": "array",
// 					"items": map[string]string{
// 						"type": "string",
// 					},
// 					"description": "List of commands to display the path or information about",
// 				},
// 			},
// 			"required": []string{"commands"},
// 		},
// 	),
// 	define("which",
// 		"locate a program file in the user's path",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"commands": map[string]interface{}{
// 					"type": "array",
// 					"items": map[string]string{
// 						"type": "string",
// 					},
// 					"description": "List of command names and searches the path for each executable file that would be run had these commands actually been invoked",
// 				},
// 			},
// 			"required": []string{"commands"},
// 		},
// 	),
// 	define("exec",
// 		"execute a command. Restrictions apply",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"command": map[string]string{
// 					"type":        "string",
// 					"description": "The command to execute",
// 				},
// 				"args": map[string]interface{}{
// 					"type": "array",
// 					"items": map[string]string{
// 						"type": "string",
// 					},
// 					"description": "optional arguments to pass to the command",
// 				},
// 			},
// 			"required": []string{"command"},
// 		},
// 	),
// 	define("env",
// 		"print environment on user's system. Only names are returned for security reasons",
// 		nil,
// 	),
// 	define("pwd",
// 		"return working directory name",
// 		nil,
// 	),
// 	define("cd",
// 		"change the current directory",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"dir": map[string]string{
// 					"type":        "string",
// 					"description": "directory to change to",
// 				},
// 			},
// 			"required": []string{"dir"},
// 		},
// 	),
// 	define("ls",
// 		"list directory contents as well as any requested, associated information",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"args": map[string]interface{}{
// 					"type": "array",
// 					"items": map[string]string{
// 						"type": "string",
// 					},
// 					"description": "files, directories and flags",
// 				},
// 			},
// 		},
// 	),
// 	define("uname",
// 		"display information about the current system's operating system and architecture",
// 		nil,
// 	),
// 	define("test",
// 		"condition evaluation utility. if it evaluates to true, returns true; otherwise it returns false.",
// 		map[string]interface{}{
// 			"type": "object",
// 			"properties": map[string]interface{}{
// 				"args": map[string]interface{}{
// 					"type": "array",
// 					"items": map[string]string{
// 						"type": "string",
// 					},
// 					"description": "flags and arguments",
// 				},
// 			},
// 			"required": []string{"args"},
// 		},
// 	),
// }

// func RunTool(cfg *internal.ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
// 	var out string
// 	var err error

// 	switch {
// 	case strings.HasPrefix(name, "ai_"):
// 		out, err = runAIHelpTool(cfg, ctx, name, props)
// 		// case strings.HasPrefix(name, "db_"):
// 		// 	out, err = runDbTool(cfg, ctx, name, props)
// 		// case strings.HasPrefix(name, "pr_"):
// 		// 	out, err = runPrTool(cfg, ctx, name, props)
// 		// case strings.HasPrefix(name, "gptr_"):
// 		// 	out, err = runGptrTool(cfg, ctx, name, props)
// 		// default:
// 		// 	out, err = runCommandTool(cfg, ctx, name, props)
// 	}

// 	if err != nil {
// 		return fmt.Sprintf("%s: %v", out, err), nil
// 	}
// 	return out, nil
// }

// func runCommandTool(cfg *internal.ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
// 	getStr := func(key string) (string, error) {
// 		return getStrProp(key, props)
// 	}
// 	getArray := func(key string) ([]string, error) {
// 		return getArrayProp(key, props)
// 	}

// 	// shell commands sensitive to the current directory or relatively "safe" to run
// 	switch name {
// 	case "exec":
// 		command, err := getStr("command")
// 		if err != nil {
// 			return "", err
// 		}
// 		args, err := getArray("args")
// 		if err != nil {
// 			return "", err
// 		}
// 		return runRestricted(ctx, cfg.Model, command, args)
// 	case "pwd":
// 		return _os.Getwd()
// 	case "cd":
// 		dir, err := getStr("dir")
// 		if err != nil {
// 			return "", err
// 		}
// 		return "", _os.Chdir(dir)
// 	case "ls":
// 		args, err := getArray("args")
// 		if err != nil {
// 			return "", err
// 		}
// 		return runCommand("ls", args)
// 	case "test":
// 		args, err := getArray("args")
// 		if err != nil {
// 			return "", err
// 		}
// 		_, err = runCommand("test", args)
// 		if err != nil {
// 			return "false", nil
// 		}
// 		return "true", nil
// 	}

// 	// Change to a temporary directory to avoid any side effects
// 	// This also magically fixes the following mysterious error for "man" (runMan):
// 	// shell-init: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
// 	// chdir: error retrieving current directory: getcwd: cannot access parent directories: No such file or directory
// 	//
// 	curDir, err := _os.Getwd()
// 	if err != nil {
// 		return "", err
// 	}
// 	defer func() {
// 		err := _os.Chdir(curDir)
// 		if err != nil {
// 			log.Errorf("runCommandTool error changing back to original directory:", err)
// 		}
// 	}()
// 	tempDir := _os.TempDir()
// 	if err := _os.Chdir(tempDir); err != nil {
// 		return "", err
// 	}

// 	switch name {
// 	case "list-commands":
// 		return ListCommandNames()
// 	case "man":
// 		command, err := getStr("command")
// 		if err != nil {
// 			return "", err
// 		}
// 		return runMan(command)
// 	case "command":
// 		commands, err := getArray("commands")
// 		if err != nil {
// 			return "", err
// 		}
// 		return runCommandV(commands)
// 	case "which":
// 		commands, err := getArray("commands")
// 		if err != nil {
// 			return "", err
// 		}
// 		return runCommand("which", commands)
// 	case "env":
// 		return _util.Env(), nil
// 	case "uname":
// 		as, arch := _util.Uname()
// 		return fmt.Sprintf("OS: %s\nArch: %s", as, arch), nil
// 	}

// 	return "", fmt.Errorf("unknown tool: %s", name)
// }

// // runCommand executes a shell command with args and returns the output
// func runCommand(command string, args []string) (string, error) {
// 	out, err := _exec.Command(command, args...).CombinedOutput()
// 	if err != nil {
// 		return fmt.Sprintf("%s %v", out, err), nil
// 	}
// 	return string(out), nil
// }

// func runMan(command string) (string, error) {
// 	manCmd := _exec.Command("man", command)
// 	var manOutput bytes.Buffer

// 	// Capture the output of the man command.
// 	manCmd.Stdout = &manOutput
// 	manCmd.Stderr = &manOutput

// 	if err := manCmd.Run(); err != nil {
// 		return "", fmt.Errorf("error running man command: %v\nOutput: %s", err, manOutput.String())
// 	}

// 	// Process with 'col' to remove formatting
// 	colCmd := _exec.Command("col", "-b")
// 	var colOutput bytes.Buffer

// 	colCmd.Stdin = bytes.NewReader(manOutput.Bytes())
// 	colCmd.Stdout = &colOutput
// 	colCmd.Stderr = &colOutput

// 	// Try running 'col', if it fails, return the man output instead.
// 	if err := colCmd.Run(); err != nil {
// 		return manOutput.String(), nil
// 	}

// 	return colOutput.String(), nil
// }

// // runCommandV executes the shell "command -v" with a list of commands and returns the output
// func runCommandV(commands []string) (string, error) {
// 	args := append([]string{"-v"}, commands...)
// 	return runCommand("command", args)
// }

// func ListCommandNames() (string, error) {
// 	list, err := _util.ListCommands(true)
// 	if err != nil {
// 		return "", err
// 	}

// 	sort.Strings(list)
// 	return strings.Join(list, "\n"), nil
// }

// func runRestricted(ctx context.Context, model *internal.Model, command string, args []string) (string, error) {
// 	// if isDenied(command) {
// 	// 	return "", fmt.Errorf("%s: Not permitted", command)
// 	// }
// 	// if isAllowed(command) {
// 	// 	return runCommand(command, args)
// 	// }

// 	// safe, err := EvaluateCommand(ctx, model, command, args)
// 	// if err != nil {
// 	// 	return "", err
// 	// }
// 	// if safe {
// 	// 	return runCommand(command, args)
// 	// }

// 	return "", fmt.Errorf("%s %s: Not permitted", command, strings.Join(args, " "))
// }

// func GetSystemTools() []openai.ChatCompletionToolParam {
// 	return systemTools
// }

// // GetRestrictedTools returns all but the exec tool
// func GetRestrictedSystemTools() []openai.ChatCompletionToolParam {
// 	exclude := []string{"exec"}
// 	return restrictedSystemTools(exclude)
// }

// func restrictedSystemTools(exclude []string) []openai.ChatCompletionToolParam {
// 	contains := func(v string) bool {
// 		for _, a := range exclude {
// 			if a == v {
// 				return true
// 			}
// 		}
// 		return false
// 	}
// 	var tools []openai.ChatCompletionToolParam
// 	for _, tool := range systemTools {
// 		name := tool.Function.Value.Name.Value
// 		if contains(name) {
// 			continue
// 		}
// 		tools = append(tools, tool)
// 	}
// 	return tools
// }
