package swarm

import (
	"encoding/json"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

type Swarm struct {
	Vars *api.Vars

	Creator api.AgentCreator
	Handler api.AgentHandler
}

// default
func New(vars *api.Vars) *Swarm {
	return &Swarm{
		Vars:    vars,
		Creator: NewAgentCreator(),
		Handler: NewAgentHandler(),
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

	ctx := req.Context()

	for {
		agent, err := r.Creator(r.Vars, req)
		if err != nil {
			return err
		}

		// dependencies
		for _, dep := range agent.Dependencies {
			depReq := &api.Request{
				Agent:    dep,
				RawInput: req.RawInput,
				// Messages: req.Messages,
			}
			depResp := &api.Response{}
			if err := r.Run(depReq, depResp); err != nil {
				return err
			}

			// decode prevous result
			// decode content as name=value and save in vars.Extra for subsequent agents
			if v, ok := r.Vars.Extra[extraResult]; ok && len(v) > 0 {
				var params = make(map[string]string)
				if err := json.Unmarshal([]byte(v), &params); err == nil {
					maps.Copy(r.Vars.Extra, params)
				}
			}

			log.GetLogger(ctx).Debug("run dependency: %s %+v\n", dep, depResp)
		}

		//
		if agent.Entrypoint != nil {
			if err := agent.Entrypoint(r.Vars, &api.Agent{
				Name:    agent.Name,
				Display: agent.Display,
			}, req.RawInput); err != nil {
				return err
			}
		}

		timeout := TimeoutHandler(r.Handler(r.Vars, agent), time.Duration(agent.MaxTime)*time.Second, "timed out")
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
