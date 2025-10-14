package vos

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// System represents the virtual operating system for the tool.
// It provides the system operations that can be mocked for testing.
type System interface {
	Man(string) (string, error)

	Command(name string, arg ...string) *exec.Cmd

	Chdir(dir string) error
	Getwd() (string, error)
}

type LocalSystem struct {
}

func NewLocalSystem() *LocalSystem {
	return &LocalSystem{}
}

func (vs *LocalSystem) Command(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}

func (vs *LocalSystem) Chdir(dir string) error {
	return os.Chdir(dir)
}

func (vs *LocalSystem) Getwd() (string, error) {
	return os.Getwd()
}

func (vs *LocalSystem) Man(s string) (string, error) {
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
