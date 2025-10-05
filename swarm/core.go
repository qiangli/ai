package swarm

import (
	"encoding/json"
	"maps"
	"os"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/vfs"
	"github.com/qiangli/ai/swarm/vos"
)

// extra result key
const extraResult = "result"

type Swarm struct {
	Vars *api.Vars

	User *api.User

	Secrets api.SecretStore
	Assets  api.AssetManager
	Tools   api.ToolSystem

	Adapters llm.AdapterRegistry

	Blobs api.BlobStore

	OS vos.System
	FS vfs.FileSystem
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

func (r *Swarm) createAgent(req *api.Request) (*api.Agent, error) {
	return conf.CreateAgent(r.Vars, r.User, r.Secrets, r.Assets, req)
}

func (r *Swarm) Run(req *api.Request, resp *api.Response) error {
	// before entering the loop clear all env
	clearAllEnv()

	ctx := req.Context()

	for {
		agent, err := r.createAgent(req)
		if err != nil {
			return err
		}

		// reset log level
		log.GetLogger(ctx).SetLogLevel(agent.LogLevel)

		// dependencies
		for _, dep := range agent.Dependencies {
			depReq := &api.Request{
				Agent:    dep,
				RawInput: req.RawInput,
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

			log.GetLogger(ctx).Debugf("Run dependency: %s %+v\n", dep, depResp)
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

		handler := NewAgentHandler(r)
		timeout := TimeoutHandler(handler(r.Vars, agent), time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)

		chain := NewChain(maxlog).Then(timeout)

		if err := chain.Serve(req, resp); err != nil {
			return err
		}

		// update the request
		if resp.Result != nil && resp.Result.State == api.StateTransfer {
			req.Agent = resp.Result.NextAgent
			continue
		}

		return nil
	}
}
