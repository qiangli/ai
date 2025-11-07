package swarm

import (
	"context"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

// global key
const globalQuery = "query"
const globalResult = "result"
const globalError = "error"

type Swarm struct {
	// TODO experimental
	Root   string
	ChatID string

	Vars *api.Vars

	User *api.User

	Secrets api.SecretStore
	Assets  api.AssetManager
	Tools   api.ToolSystem

	Adapters llm.AdapterRegistry

	Blobs api.BlobStore

	OS        vos.System
	Workspace vfs.Workspace

	History api.MemStore
}

func (r *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	return conf.CreateAgent(ctx, req, r.User, r.Secrets, r.Assets)
}

// Run calls the language model with the messages list (after applying the system prompt). If the resulting AIMessage contains tool_calls, the graph will then call the tools. The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (r *Swarm) Run(req *api.Request, resp *api.Response) error {
	if req.Name == "" {
		return api.NewBadRequestError("missing agent in request")
	}
	if req.RawInput == nil {
		return api.NewBadRequestError("missing raw input in request")
	}

	var ctx = req.Context()
	var resetLogLevel = true

	if r.User == nil || r.Vars == nil {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}
	agentHandler, err := NewAgentHandler(r)
	if err != nil {
		return err
	}

	log.GetLogger(ctx).Debugf("*** Agent: %s parent: %+v\n", req.Name, req.Parent)

	for {
		start := time.Now()
		log.GetLogger(ctx).Debugf("Creating agent: %s %s\n", req.Name, start)

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

		timeout := TimeoutHandler(agentHandler(agent), time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)
		chain := NewChain(maxlog)
		if err := chain.Then(timeout).Serve(req, resp); err != nil {
			return err
		}

		// update the request
		if resp.Result != nil && resp.Result.State == api.StateTransfer {
			log.GetLogger(ctx).Debugf("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
			req.Name = resp.Result.NextAgent
			continue
		}

		end := time.Now()
		log.GetLogger(ctx).Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}
