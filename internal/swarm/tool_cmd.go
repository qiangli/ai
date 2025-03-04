package swarm

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/llm/vos"
	"github.com/qiangli/ai/internal/log"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os
var _util = _os

func CallCommandTool(ctx context.Context, agent *Agent, name string, props map[string]any) (string, error) {
	getStr := func(key string) (string, error) {
		return GetStrProp(key, props)
	}
	getArray := func(key string) ([]string, error) {
		return GetArrayProp(key, props)
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
		return runRestricted(ctx, agent, command, args)
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
		return runCommand("ls", args)
	case "test":
		args, err := getArray("args")
		if err != nil {
			return "", err
		}
		_, err = runCommand("test", args)
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
	case "list_commands":
		return listCommandNames()
	case "man":
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		return runMan(command)
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
		return runCommand("which", commands)
	case "env":
		return _util.Env(), nil
	case "uname":
		as, arch := _util.Uname()
		return fmt.Sprintf("OS: %s\nArch: %s", as, arch), nil
	}

	return "", fmt.Errorf("unknown tool: %s", name)
}

// runCommand executes a shell command with args and returns the output
func runCommand(command string, args []string) (string, error) {
	out, err := _exec.Command(command, args...).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s %v", out, err), nil
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

// runCommandV executes the shell "command -v" with a list of commands and returns the output
func runCommandV(commands []string) (string, error) {
	args := append([]string{"-v"}, commands...)
	return runCommand("command", args)
}

func listCommandNames() (string, error) {
	list, err := _util.ListCommands(true)
	if err != nil {
		return "", err
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), nil
}

func runRestricted(ctx context.Context, agent *Agent, command string, args []string) (string, error) {
	if isDenied(command) {
		return "", fmt.Errorf("%s: Not permitted", command)
	}
	if isAllowed(command) {
		return runCommand(command, args)
	}

	safe, err := evaluateCommand(ctx, agent, command, args)
	if err != nil {
		return "", err
	}
	if safe {
		return runCommand(command, args)
	}

	return "", fmt.Errorf("%s %s: Not permitted", command, strings.Join(args, " "))
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
