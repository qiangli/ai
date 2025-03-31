package vos

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/util"
)

// System represents the virtual operating system for the tool.
// It provides the system operations that can be mocked for testing.
type System interface {
	ListCommands() []string
	Which([]string) (string, error)
	Man(string) (string, error)

	Command(name string, arg ...string) *exec.Cmd

	Chdir(dir string) error
	Getwd() (string, error)

	Env() string
	Uname() (string, string)
}

type VirtualSystem struct {
}

func NewSystem() *VirtualSystem {
	return &VirtualSystem{}
}

func (vs *VirtualSystem) ListCommands() []string {
	list := util.ListCommands()
	var commands []string
	for k := range list {
		commands = append(commands, k)
	}
	sort.Strings(commands)
	return commands
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

func (vs *VirtualSystem) Man(s string) (string, error) {
	command := strings.TrimSpace(strings.SplitN(s, " ", 2)[0])
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
