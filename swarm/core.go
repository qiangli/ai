package swarm

import (
	"fmt"
	"os"
	"strings"
	"time"

	// "github.com/qiangli/ai/internal/util"
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
		// agent, err := CreateAgent(r.Vars, req.Agent, req.Command, req.RawInput)
		if r.Vars == nil || r.Vars.Config == nil || r.Vars.Config.AgentCreator == nil {
			return fmt.Errorf("not initialized.")
		}
		agent, err := r.Vars.Config.AgentCreator(r.Vars, req)
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
