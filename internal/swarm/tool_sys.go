package swarm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/swarm/vfs"
	"github.com/qiangli/ai/internal/swarm/vos"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os

func ListSystemTools() ([]*ToolFunc, error) {
	var tools []*ToolFunc
	for k, v := range vfs.Descriptors {
		tools = append(tools, &ToolFunc{
			Name:        k,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

	for k, v := range vos.Descriptors {
		tools = append(tools, &ToolFunc{
			Name:        k,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

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

func sortTools(tools []*ToolFunc) {
	for i := range len(tools) - 1 {
		for j := i + 1; j < len(tools); j++ {
			if tools[i].Name > tools[j].Name {
				tools[i], tools[j] = tools[j], tools[i]
			}
		}
	}
}

func CallSystemTool(fs vfs.FileSystem, ctx context.Context, agent *Agent, name string, props map[string]any) (string, error) {

	getStr := func(key string) (string, error) {
		return GetStrProp(key, props)
	}
	getArray := func(key string) ([]string, error) {
		return GetArrayProp(key, props)
	}

	// vfs
	switch name {
	case vfs.ListRootsToolName:
		roots, err := fs.ListRoots()
		if err != nil {
			return "", err
		}
		return strings.Join(roots, "\n"), nil
	case vfs.ListDirectoryToolName:
		path, err := getStr("path")
		if err != nil {
			return "", err
		}
		list, err := fs.ListDirectory(path)
		if err != nil {
			return "", err
		}
		return strings.Join(list, "\n"), nil
	case vfs.CreateDirectoryToolName:
		path, err := getStr("path")
		if err != nil {
			return "", err
		}
		return "", fs.CreateDirectory(path)
	case vfs.RenameFileToolName:
		source, err := getStr("source")
		if err != nil {
			return "", err
		}
		dest, err := getStr("destination")
		if err != nil {
			return "", err
		}
		if err := fs.RenameFile(source, dest); err != nil {
			return "", err
		}
		return "File renamed successfully", nil
	case vfs.GetFileInfoToolName:
		path, err := getStr("path")
		if err != nil {
			return "", err
		}
		info, err := fs.GetFileInfo(path)
		if err != nil {
			return "", err
		}
		return info.String(), nil
	case vfs.ReadFileToolName:
		path, err := getStr("path")
		if err != nil {
			return "", err
		}
		content, err := fs.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	case vfs.WriteFileToolName:
		path, err := getStr("path")
		if err != nil {
			return "", err
		}
		content, err := getStr("content")
		if err != nil {
			return "", err
		}
		if err := fs.WriteFile(path, []byte(content)); err != nil {
			return "", err
		}
		return "File written successfully", nil
	case vfs.TempDirToolName:
		return fs.TempDir(), nil
	}

	// vos
	switch name {
	case vos.ListCommandsToolName:
		list, err := _os.ListCommands()
		if err != nil {
			return "", err
		}
		return strings.Join(list, "\n"), nil
	case vos.WhichToolName:
		commands, err := getArray("commands")
		if err != nil {
			return "", err
		}
		return _os.Which(commands)
	case vos.ManToolName:
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		return _os.Man(command)
	case vos.ExecToolName:
		command, err := getStr("command")
		if err != nil {
			return "", err
		}
		args, err := getArray("args")
		if err != nil {
			return "", err
		}
		return runRestricted(fs, ctx, agent, command, args)
	case vos.CdToolName:
		dir, err := getStr("dir")
		if err != nil {
			return "", err
		}
		return "", _os.Chdir(dir)
	case vos.PwdToolName:
		return _os.Getwd()
	case vos.EnvToolName:
		return _os.Env(), nil
	case vos.UnameToolName:
		as, arch := _os.Uname()
		return fmt.Sprintf("OS: %s\nArch: %s", as, arch), nil
	}

	// misc
	switch name {
	case ReadStdinToolName:
		return readStdin()
	case ReadClipboardToolName:
		return readClipboard()
	case ReadClipboardWaitToolName:
		return readClipboardWait()
	case WriteStdoutToolName:
		content, err := getStr("content")
		if err != nil {
			return "", err
		}
		return writeStdout(content)
	case WriteClipboardToolName:
		content, err := getStr("content")
		if err != nil {
			return "", err
		}
		return writeClipboard(content)
	case WriteClipboardAppendToolName:
		content, err := getStr("content")
		if err != nil {
			return "", err
		}
		return writeClipboardAppend(content)
	case GetUserTextInputToolName:
		prompt, err := getStr("prompt")
		if err != nil {
			return "", err
		}
		return getUserTextInput(prompt)
	case GetUserChoiceInputToolName:
		prompt, err := getStr("prompt")
		if err != nil {
			return "", err
		}
		choices, err := getArray("choices")
		if err != nil {
			return "", err
		}
		defaultChoice, err := getStr("default")
		if err != nil {
			return "", err
		}
		return getUserChoiceInput(prompt, choices, defaultChoice)
	}

	return "", fmt.Errorf("unknown system tool: %s", name)
}

// runCommand executes a shell command with args and returns the output
func runCommand(command string, args []string) (string, error) {
	out, err := _exec.Command(command, args...).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s %v", out, err), nil
	}
	return string(out), nil
}

// runCommandV executes the shell "command -v" with a list of commands and returns the output
func runCommandV(commands []string) (string, error) {
	args := append([]string{"-v"}, commands...)
	return runCommand("command", args)
}

func runRestricted(fs vfs.FileSystem, ctx context.Context, agent *Agent, command string, args []string) (string, error) {
	if isDenied(command) {
		return "", fmt.Errorf("%s: Not permitted", command)
	}
	if isAllowed(command) {
		return runCommand(command, args)
	}

	safe, err := evaluateCommand(fs, ctx, agent, command, args)
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
