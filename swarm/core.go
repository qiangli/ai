package swarm

import (
	"context"
	// "encoding/json"
	// "maps"
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

// global key
const globalQuery = "query"
const globalResult = "result"

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

	History api.MemStore
}

func (s *Swarm) Clone() *Swarm {
	return &Swarm{
		Vars: s.Vars.Clone(),
		//
		User: s.User,
		//
		Secrets: s.Secrets,
		Assets:  s.Assets,
		Tools:   s.Tools,
		//
		Adapters: s.Adapters,
		//
		Blobs: s.Blobs,
		//
		OS: s.OS,
		FS: s.FS,
		//
		History: s.History,
	}
}

// Function to clear all environment variables execep essential ones
func ClearAllEnv() {
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

func (r *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	return conf.CreateAgent(ctx, r.Vars, r.User, req.Agent, req.RawInput, r.Secrets, r.Assets)
}

func (r *Swarm) Run(req *api.Request, resp *api.Response) error {
	if req.Agent == "" {
		return api.NewBadRequestError("missing agent in request")
	}
	if req.RawInput == nil {
		return api.NewBadRequestError("missing raw input in request")
	}

	var ctx = req.Context()
	var resetLogLevel = true

	agentHandler := NewAgentHandler(r)

	log.GetLogger(ctx).Debugf("*** Agent: %s parent: %+v\n", req.Agent, req.Parent)

	for {
		start := time.Now()
		log.GetLogger(ctx).Debugf("Creating agent: %s %s\n", req.Agent, start)

		//
		agent, err := r.createAgent(ctx, req)
		if err != nil {
			return err
		}

		// reset log level
		// subsequent agents will inhefit the same log level
		if resetLogLevel {
			resetLogLevel = false
			log.GetLogger(ctx).SetLogLevel(agent.LogLevel)
		}

		// // dependencies
		// for _, dep := range agent.Dependencies {
		// 	depReq := &api.Request{
		// 		Agent:    dep,
		// 		RawInput: req.RawInput,
		// 	}
		// 	depResp := &api.Response{}
		// 	if err := r.Run(depReq, depResp); err != nil {
		// 		return err
		// 	}

		// 	// decode prevous result
		// 	// decode content as name=value and save in vars.Extra for subsequent agents
		// 	if v, ok := r.Vars.Extra[extraResult]; ok && len(v) > 0 {
		// 		var params = make(map[string]string)
		// 		if err := json.Unmarshal([]byte(v), &params); err == nil {
		// 			maps.Copy(r.Vars.Extra, params)
		// 		}
		// 	}

		// 	log.GetLogger(ctx).Debugf("dependency complete: %s %+v\n", dep, depResp)
		// }

		// //
		// if agent.Entrypoint != nil {
		// 	if err := agent.Entrypoint(r.Vars, &api.Agent{
		// 		Name:    agent.Name,
		// 		Display: agent.Display,
		// 	}, req.RawInput); err != nil {
		// 		return err
		// 	}
		// }

		timeout := TimeoutHandler(agentHandler(r.Vars, agent), time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)
		chain := NewChain(maxlog)
		if err := chain.Then(timeout).Serve(req, resp); err != nil {
			return err
		}

		// update the request
		if resp.Result != nil && resp.Result.State == api.StateTransfer {
			log.GetLogger(ctx).Debugf("Agent transfer: %s => %s\n", req.Agent, resp.Result.NextAgent)
			req.Agent = resp.Result.NextAgent
			continue
		}

		end := time.Now()
		log.GetLogger(ctx).Debugf("Agent complete: %s %s elapsed: %s\n", req.Agent, end, end.Sub(start))
		return nil
	}
}
