package swarm

import (
	"context"
	"fmt"
	"maps"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"

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

	// TODO
	template *template.Template
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

	var fm = sprig.FuncMap()
	maps.Copy(fm, tplFuncMap)
	fm["user"] = func() *api.User {
		return r.User
	}
	r.template = template.New("swarm").Funcs(fm)

	logMiddleware := MaxLogMiddlewareFunc(r)
	envMiddleware := EnvMiddlewareFunc(r)
	memMiddleware := MemoryMiddlewareFunc(r)
	modelMiddleware := ModelMiddlewareFunc(r)
	agentMiddlWare := AgentMiddlewareFunc(r)

	log.GetLogger(ctx).Debugf("*** Agent: %s parent: %+v\n", req.Name, req.Parent)

	final := HandlerFunc(func(req *api.Request, res *api.Response) error {
		return nil
	})

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

		chain := NewChain(
			TimeoutMiddleware(agent.MaxTime),
			logMiddleware(agent, 100),
			//
			envMiddleware(agent),
			memMiddleware(agent),
			modelMiddleware(agent),
			agentMiddlWare(agent),
		)

		if err := chain.Then(final).Serve(req, resp); err != nil {
			return err
		}

		if resp.Result == nil {
			// some thing went wrong
			return fmt.Errorf("nil result running %q", agent.Name)
		}

		if resp.Result.State == api.StateTransfer {
			log.GetLogger(ctx).Debugf("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
			req.Name = resp.Result.NextAgent
			continue
		}

		end := time.Now()
		log.GetLogger(ctx).Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}
