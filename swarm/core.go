package swarm

import (
	"os"
	"strings"
	"time"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
)

type Swarm struct {
	Vars *api.Vars
}

func New(vars *api.Vars) *Swarm {
	return &Swarm{
		Vars: vars,
	}
}

func InitVars(app *api.AppConfig) (*api.Vars, error) {
	agentLoader, err := initAgents(app)
	if err != nil {
		return nil, err
	}
	app.AgentLoader = agentLoader

	toolLoader, err := initTools(app)
	if err != nil {
		return nil, err
	}
	app.ToolLoader = toolLoader

	modelLoader, err := initModels(app, app.Models)
	if err != nil {
		return nil, err
	}
	app.ModelLoader = modelLoader

	return Vars(app)
}

func Vars(app *api.AppConfig) (*api.Vars, error) {

	var vars = api.NewVars()
	//
	vars.Config = app

	//
	vars.Workspace = app.Workspace
	// vars.Repo = app.Repo
	vars.Home = app.Home
	vars.Temp = app.Temp

	//
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return nil, err
	}

	vars.Arch = sysInfo.Arch
	vars.OS = sysInfo.OS
	vars.ShellInfo = sysInfo.ShellInfo
	vars.OSInfo = sysInfo.OSInfo
	vars.UserInfo = sysInfo.UserInfo

	//
	// vars.TemplateFuncMap = tplFuncMap
	// vars.AdviceMap = adviceMap

	return vars, nil
}

// Function to clear all environment variables execep essential ones
func clearAllEnv() {
	essentialEnv := []string{"PATH", "PWD", "HOME", "USER", "SHELL"}

	essentialMap := make(map[string]bool, len(essentialEnv))
	for _, key := range essentialEnv {
		essentialMap[key] = true
	}

	for _, env := range os.Environ() {
		key := strings.Split(env, "=")[0]
		if !essentialMap[key] {
			os.Unsetenv(key)
		}
	}
}

func (r *Swarm) Run(req *api.Request, resp *api.Response) error {
	// before entering the loop clear all env
	clearAllEnv()

	for {
		agent, err := CreateAgent(r.Vars, req.Agent, req.Command, req.RawInput)
		if err != nil {
			return err
		}

		if agent.Entrypoint != nil {
			if err := agent.Entrypoint(r.Vars, &api.Agent{
				Name:    agent.Name,
				Display: agent.Display,
			}, req.RawInput); err != nil {
				return err
			}
		}

		timeout := TimeoutHandler(AgentHandler(r.Vars, agent), time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)

		chain := NewChain(maxlog).Then(timeout)

		if err := chain.Serve(req, resp); err != nil {
			return err
		}

		// update the request
		result := resp.Result
		if result != nil && result.State == api.StateTransfer {
			req.Agent = result.NextAgent
			continue
		}
		return nil
	}
}
