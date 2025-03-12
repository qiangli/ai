package vos

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/qiangli/ai/internal/util"
)

// System represents the virtual operating system for the tool.
// It provides the system operations that can be mocked for testing.
type System interface {
	ListCommands() ([]string, error)
	Which([]string) (string, error)
	Man(string) (string, error)

	Command(name string, arg ...string) *exec.Cmd

	Chdir(dir string) error
	Getwd() (string, error)

	Env() string
	Uname() (string, string)
}

// type Descriptor struct {
// 	Name        string
// 	Description string
// 	Parameters  map[string]any
// }

const (
	ListCommandsToolName = "list_commands"
	WhichToolName        = "which"
	ManToolName          = "man"
	ExecToolName         = "exec"
	CdToolName           = "cd"
	PwdToolName          = "pwd"
	EnvToolName          = "env"
	UnameToolName        = "uname"
)

// var Descriptors = map[string]*Descriptor{
// 	ListCommandsToolName: {
// 		Name:        ListCommandsToolName,
// 		Description: "List all available command names on the user's path. Use 'which' to get the full path",
// 		Parameters:  map[string]any{},
// 	},
// 	WhichToolName: {
// 		Name:        WhichToolName,
// 		Description: "Locate a program file on the user's path",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"commands": map[string]any{
// 					"type": "array",
// 					"items": map[string]any{
// 						"type": "string",
// 					},
// 					"description": "List of command names and searches the path for each executable file that would be run had these commands actually been invoked",
// 				},
// 			},
// 			"required": []string{"commands"},
// 		},
// 	},
// 	ManToolName: {
// 		Name:        ManToolName,
// 		Description: "Find and display online manual documentation page for a command. Not all commands have manual pages",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"command": map[string]any{
// 					"type":        "string",
// 					"description": "The command to get the manual page for",
// 				},
// 			},
// 			"required": []string{"command"},
// 		},
// 	},
// 	ExecToolName: {
// 		Name:        ExecToolName,
// 		Description: "Execute a command in the user's environment. Restrictions may apply due to security",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"command": map[string]any{
// 					"type":        "string",
// 					"description": "The command to execute",
// 				},
// 				"args": map[string]any{
// 					"type":        "array",
// 					"items":       map[string]any{"type": "string"},
// 					"description": "The arguments to pass to the command. may be empty",
// 				},
// 			},
// 			"required": []string{"command"},
// 		},
// 	},
// 	CdToolName: {
// 		Name:        CdToolName,
// 		Description: "Change the current working directory on user's system",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"dir": map[string]any{
// 					"type":        "string",
// 					"description": "The directory to change to",
// 				},
// 			},
// 			"required": []string{"dir"},
// 		},
// 	},
// 	PwdToolName: {
// 		Name:        PwdToolName,
// 		Description: "Print the current working directory on user's system",
// 		Parameters:  map[string]any{},
// 	},
// 	EnvToolName: {
// 		Name:        EnvToolName,
// 		Description: "Print environment variables on user's system. Only names are returned for security reasons",
// 		Parameters:  map[string]any{},
// 	},
// 	UnameToolName: {
// 		Name:        UnameToolName,
// 		Description: "Display information about the current system's operating system and architecture",
// 		Parameters:  map[string]any{},
// 	},
// }

type VirtualSystem struct {
}

func NewVirtualSystem() *VirtualSystem {
	return &VirtualSystem{}
}

func (vs *VirtualSystem) ListCommands() ([]string, error) {
	list, err := util.ListCommands(true)
	if err != nil {
		return nil, err
	}

	sort.Strings(list)
	return list, nil
}

func (vs *VirtualSystem) Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func (vs *VirtualSystem) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (vs *VirtualSystem) Getwd() (string, error) {
	return os.Getwd()
}

func (vs *VirtualSystem) Env() string {
	return util.GetEnvVarNames()
}

func (vs *VirtualSystem) Uname() (string, string) {
	return util.Uname()
}

func (vs *VirtualSystem) Man(command string) (string, error) {
	manCmd := vs.Command("man", command)
	var manOutput bytes.Buffer

	// Capture the output of the man command.
	manCmd.Stdout = &manOutput
	manCmd.Stderr = &manOutput

	if err := manCmd.Run(); err != nil {
		return "", fmt.Errorf("error running man command: %v\nOutput: %s", err, manOutput.String())
	}

	// Process with 'col' to remove formatting
	colCmd := vs.Command("col", "-b")
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

func (vs *VirtualSystem) Which(args []string) (string, error) {
	out, err := vs.Command("which", args...).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s %v", out, err), nil
	}
	return string(out), nil
}
