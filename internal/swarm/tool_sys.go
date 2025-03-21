package swarm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qiangli/ai/internal/swarm/vfs"
	"github.com/qiangli/ai/internal/swarm/vos"
	"github.com/qiangli/ai/internal/util"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os

var FSDescriptors = map[string]*Descriptor{
	// vfs.ListRootsToolName: {
	// 	Name:        vfs.ListRootsToolName,
	// 	Description: "Returns the list of directories that this server is allowed to access.",
	// 	Parameters: map[string]any{
	// 		"type":       "object",
	// 		"properties": map[string]any{},
	// 	},
	// },
	vfs.ListDirectoryToolName: {
		Name:        vfs.ListDirectoryToolName,
		Description: "Get a detailed listing of all files and directories in a specified path.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path of the directory to list",
				},
			},
			"required": []string{"path"},
		},
	},
	vfs.CreateDirectoryToolName: {
		Name:        vfs.CreateDirectoryToolName,
		Description: "Create a new directory or ensure a directory exists.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path of the directory to create",
				},
			},
			"required": []string{"path"},
		},
	},
	vfs.RenameFileToolName: {
		Name:        vfs.RenameFileToolName,
		Description: "Rename files and directories.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"source": map[string]any{
					"type":        "string",
					"description": "Source path of the file or directory",
				},
				"destination": map[string]any{
					"type":        "string",
					"description": "Destination path",
				},
			},
			"required": []string{"source", "destination"},
		},
	},
	vfs.GetFileInfoToolName: {
		Name:        vfs.GetFileInfoToolName,
		Description: "Retrieve detailed metadata about a file or directory.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file or directory",
				},
			},
			"required": []string{"path"},
		},
	},
	vfs.ReadFileToolName: {
		Name:        vfs.ReadFileToolName,
		Description: "Read the complete contents of a file from the file system.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path to the file to read",
				},
			},
			"required": []string{"path"},
		},
	},
	vfs.WriteFileToolName: {
		Name:        vfs.WriteFileToolName,
		Description: "Create a new file or overwrite an existing file with new content.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Path where to write the file",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Content to write to the file",
				},
			},
			"required": []string{"path", "content"},
		},
	},
}

var OSDescriptors = map[string]*Descriptor{
	vos.ListCommandsToolName: {
		Name:        vos.ListCommandsToolName,
		Description: "List all available command names on the user's path. Use 'which' to get the full path",
		Parameters:  map[string]any{},
	},
	vos.WhichToolName: {
		Name:        vos.WhichToolName,
		Description: "Locate a program file on the user's path",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"commands": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "List of command names and searches the path for each executable file that would be run had these commands actually been invoked",
				},
			},
			"required": []string{"commands"},
		},
	},
	vos.ManToolName: {
		Name:        vos.ManToolName,
		Description: "Find and display online manual documentation page for a command. Not all commands have manual pages",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The command to get the manual page for",
				},
			},
			"required": []string{"command"},
		},
	},
	vos.ExecToolName: {
		Name:        vos.ExecToolName,
		Description: "Executes a specified command within the user's environment, allowing optional flags and arguments to be passed via 'args'. Note: some security restrictions may apply.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "The command to execute",
				},
				"args": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "The arguments to pass to the command. may be empty",
				},
			},
			"required": []string{"command"},
		},
	},
	vos.CdToolName: {
		Name:        vos.CdToolName,
		Description: "Change the current working directory on user's system",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dir": map[string]any{
					"type":        "string",
					"description": "The directory to change to",
				},
			},
			"required": []string{"dir"},
		},
	},
	vos.PwdToolName: {
		Name:        vos.PwdToolName,
		Description: "Print the current working directory on user's system",
		Parameters:  map[string]any{},
	},
	vos.EnvToolName: {
		Name:        vos.EnvToolName,
		Description: "Print environment variables on user's system. Only names are returned for security reasons",
		Parameters:  map[string]any{},
	},
	vos.UnameToolName: {
		Name:        vos.UnameToolName,
		Description: "Display information about the current system's operating system and architecture",
		Parameters:  map[string]any{},
	},
}

const (
	HostHomeDirToolName   = "home_dir"
	HostTempDirToolName   = "temp_dir"
	HostRepoDirToolName   = "repo_dir"
	HostWorkspaceToolName = "workspace"
)

var HostDescriptors = map[string]*Descriptor{
	HostHomeDirToolName: {
		Name:        HostHomeDirToolName,
		Description: "Return the user's home directory",
		Parameters:  map[string]any{},
	},
	HostTempDirToolName: {
		Name:        HostTempDirToolName,
		Description: "Return the default directory to use for temporary files",
		Parameters:  map[string]any{},
	},
	HostWorkspaceToolName: {
		Name:        HostWorkspaceToolName,
		Description: "Return the current workspace directory",
		Parameters:  map[string]any{},
	},
	HostRepoDirToolName: {
		Name:        HostRepoDirToolName,
		Description: "Return the current repository directory",
		Parameters:  map[string]any{},
	},
}

func workDir() (string, error) {
	return _os.Getwd()
}

func homeDir() (string, error) {
	return os.UserHomeDir()
}

func tempDir() (string, error) {
	return os.TempDir(), nil
}

// default to repo dir if workspace is not provided
func resolveWorkspaceDir(ws string) (string, error) {
	if ws != "" {
		return util.ResolveWorkspace(ws)
	}
	return resolveRepoDir()
}

// resolveRepoDir returns the directory of the current git repository
func resolveRepoDir() (string, error) {
	wd, err := workDir()
	if err != nil {
		return "", err
	}
	dir, err := util.DetectGitRepo(wd)
	if err != nil {
		return "", fmt.Errorf("failed to detect git repository: %w", err)
	}
	return dir, nil
}

var systemTools []*ToolFunc

func init() {
	var tools []*ToolFunc
	for _, v := range FSDescriptors {
		tools = append(tools, &ToolFunc{
			Label:       ToolLabelSystem,
			Service:     "fs",
			Func:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

	for _, v := range OSDescriptors {
		tools = append(tools, &ToolFunc{
			Label:       ToolLabelSystem,
			Service:     "os",
			Func:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

	for _, v := range MiscDescriptors {
		tools = append(tools, &ToolFunc{
			Label:       ToolLabelSystem,
			Service:     "misc",
			Func:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

	// host tools
	for _, v := range HostDescriptors {
		tools = append(tools, &ToolFunc{
			Label:       ToolLabelSystem,
			Service:     "host",
			Func:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
		})
	}

	systemTools = tools
}

func ListSystemTools() ([]*ToolFunc, error) {
	return systemTools, nil
}

func callSystemTool(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	getStr := func(key string) (string, error) {
		return GetStrProp(key, args)
	}
	getArray := func(key string) ([]string, error) {
		return GetArrayProp(key, args)
	}

	v, ok := vars.ToolMap[name]
	if !ok {
		return "", fmt.Errorf("no such system tool: %s", name)
	}

	// vfs
	if v.Service == "fs" {
		fs := vars.FS
		switch v.Func {
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
		}
		return "", fmt.Errorf("unknown file system tool: %s", name)
	}

	// vos
	if v.Service == "os" {
		switch v.Func {
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
			return runRestricted(ctx, vars, command, args)
		case vos.CdToolName:
			dir, err := getStr("dir")
			if err != nil {
				return "", err
			}
			return "", _os.Chdir(dir)
		case vos.PwdToolName:
			return workDir()
		case vos.EnvToolName:
			return _os.Env(), nil
		case vos.UnameToolName:
			as, arch := _os.Uname()
			return fmt.Sprintf("OS: %s\nArch: %s", as, arch), nil
		}
		return "", fmt.Errorf("unknown os tool: %s", name)
	}

	// host
	if v.Service == "host" {
		switch v.Func {
		case HostHomeDirToolName:
			return homeDir()
		case HostTempDirToolName:
			return tempDir()
		case HostWorkspaceToolName:
			return resolveWorkspaceDir(vars.Workspace)
		case HostRepoDirToolName:
			return resolveRepoDir()
		}
		return "", fmt.Errorf("unknown host tool: %s", name)
	}

	// misc
	if v.Service == "misc" {
		switch v.Func {
		case ReadStdinToolName:
			return readStdin()
		case PasteFromClipboardToolName:
			return readClipboard()
		case PasteFromClipboardWaitToolName:
			return readClipboardWait()
		case WriteStdoutToolName:
			content, err := getStr("content")
			if err != nil {
				return "", err
			}
			return writeStdout(content)
		case CopyToClipboardToolName:
			content, err := getStr("content")
			if err != nil {
				return "", err
			}
			return writeClipboard(content)
		case CopyToClipboardAppendToolName:
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
		return "", fmt.Errorf("unknown misc tool: %s", name)
	}

	return "", fmt.Errorf("unknown service: %s", name)
}

// runCommand executes a shell command with args and returns the output
func runCommand(command string, args []string) (string, error) {
	var out []byte
	var err error
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		out, err = _exec.Command("sh", "-c", command).CombinedOutput()
	} else {
		out, err = _exec.Command(command, args...).CombinedOutput()
	}
	if err != nil {
		// send error with out to assist LLM
		return "", fmt.Errorf("%s %v: %v\n %s", command, args, err, out)
	}
	return string(out), nil
}

func runRestricted(ctx context.Context, vars *Vars, command string, args []string) (string, error) {
	if isDenied(command) {
		return "", fmt.Errorf("%s: Not allowed", command)
	}
	if isAllowed(command) {
		return runCommand(command, args)
	}

	safe, err := evaluateCommand(ctx, vars, command, args)
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
