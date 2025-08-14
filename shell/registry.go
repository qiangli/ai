package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
)

var commandRegistry map[string]string

// var agentRegistry map[string]*api.AgentConfig
var aliasRegistry map[string]string

var visitedRegistry *VisitedRegistry

func initRegistry(vars *api.Vars) (err error) {
	commandRegistry = util.ListCommands()
	// agentRegistry = vars.ListAgents()
	aliasRegistry, err = listAlias(vars.Config.Shell)
	if err != nil {
		return err
	}
	visitedRegistry, err = NewVisitedRegistry()
	if err != nil {
		return err
	}

	return nil
}

type VisitedRegistry struct {
	home string

	visited map[string]bool
}

func (r *VisitedRegistry) Visit(abs string) {
	if abs == "" {
		return
	}
	if abs == r.home {
		return
	}
	rel := strings.TrimPrefix(abs, r.home)
	if rel == "" {
		return
	}
	var visited string
	if abs == rel {
		visited = abs
	} else {
		rel = strings.TrimPrefix(rel, string(filepath.Separator))
		if rel == "" {
			return
		}
		visited = fmt.Sprintf("~%s%s", string(filepath.Separator), rel)
	}
	r.visited[visited] = true
}

func (r *VisitedRegistry) List() []string {
	var list []string
	for k := range r.visited {
		list = append(list, k)
	}
	return list
}

func NewVisitedRegistry() (*VisitedRegistry, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		return nil, err
	}

	reg := &VisitedRegistry{
		home:    abs,
		visited: make(map[string]bool),
	}

	// init with current working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	reg.Visit(wd)

	return reg, nil
}

// Chdir changes the current working directory to the specified path.
// This is required to update the PWD environment variable
func Chdir(dir string) error {
	log.Debugf("chdir to: %s\n", dir)

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.Chdir(abs); err != nil {
		return err
	}
	if err := os.Setenv("PWD", abs); err != nil {
		return err
	}

	visitedRegistry.Visit(abs)

	log.Debugf("chdir changed to: %s\n", abs)
	return nil
}
