package vos

import (
	"os"
	"os/exec"

	"github.com/qiangli/ai/internal/util"
)

// System represents the virtual operating system for the tool.
// It provides the system operations that can be mocked for testing.
type System interface {
	Command(name string, arg ...string) *exec.Cmd
	Chdir(dir string) error
	Getwd() (string, error)
	MkdirAll(path string, perm os.FileMode) error
	TempDir() string

	//
	Env() string
	Uname() (string, string)
	ListCommands(nameOnly bool) ([]string, error)
}

type VirtualSystem struct {
}

func NewVirtualSystem() *VirtualSystem {
	return &VirtualSystem{}
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

func (vs *VirtualSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (vs *VirtualSystem) TempDir() string {
	return os.TempDir()
}

func (vs *VirtualSystem) Env() string {
	return util.GetEnvVarNames()
}

func (vs *VirtualSystem) Uname() (string, string) {
	return util.Uname()
}

func (vs *VirtualSystem) ListCommands(nameOnly bool) ([]string, error) {
	return util.ListCommands(nameOnly)
}
