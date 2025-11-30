package swarm

import (
	"context"
	// "encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/u-root/u-root/pkg/shlex"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

// global key
// const globalQuery = "query"
// const globalResult = "result"
// const globalError = "error"

type Swarm struct {
	// swarm session id
	ID string

	Vars *api.Vars

	User *api.User

	Secrets api.SecretStore

	Assets api.AssetManager

	Tools api.ToolSystem

	Adapters llm.AdapterRegistry

	Blobs api.BlobStore

	// virtual system
	Root      string
	OS        vos.System
	Workspace vfs.Workspace

	History api.MemStore

	// internal
	middlewares []api.Middleware
	agentMaker  *AgentMaker
}

func (sw *Swarm) Init() {
	sw.InitChain()
	sw.agentMaker = NewAgentMaker(sw)
	//
}

func (sw *Swarm) InitChain() {
	// logging, analytics, and debugging.
	// prompts, tool selection, and output formatting.
	// retries, fallbacks, early termination.
	// rate limits, guardrails, pii detection.
	sw.middlewares = []api.Middleware{
		InitEnvMiddleware(sw),

		// cross cutting
		TimeoutMiddleware(sw),
		LogMiddleware(sw),

		//
		ModelMiddleware(sw),
		ToolMiddleware(sw),
		//
		InstructionMiddleware(sw),
		ContextMiddleware(sw),
		QueryMiddleware(sw),
		//
		AgentFlowMiddleware(sw),

		//
		InferenceMiddleware(sw),
		// output
	}
}

func (sw *Swarm) CreateAgent(ctx context.Context, name string) (*api.Agent, error) {
	if name == "" {
		// anonymous
		log.GetLogger(ctx).Debugf("agent not specified.\n")
	}
	agent, err := sw.agentMaker.CreateAgent(ctx, name)

	if err != nil {
		return nil, err
	}

	// for sub agent/action or tool call
	agent.Runner = NewAgentToolRunner(sw, agent)
	agent.Template = NewTemplate(sw, agent)
	return agent, nil
}

// Run calls the language model with the messages list (after applying the system prompt).
// If the resulting AI Message contains tool_calls, the orchestrator will then call the tools.
// The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (sw *Swarm) Run(req *api.Request, resp *api.Response) error {
	if sw.User == nil || sw.Vars == nil {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}
	if v, _ := sw.Vars.Global.Get("workspace"); v == "" {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}
	if req.Agent != nil && req.Agent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself not supported.", req.Name))
	}

	var ctx = req.Context()
	logger := log.GetLogger(ctx)

	for {
		start := time.Now()
		logger.Debugf("creating agent: %s %s\n", req.Name, start)

		// creator
		agent, err := sw.CreateAgent(ctx, req.Name)
		if err != nil {
			return err
		}
		// inherit args
		var addAll func(*api.Agent)
		addAll = func(a *api.Agent) {
			if a == nil {
				return
			}
			if a.Parent != nil {
				addAll(a.Parent)
			}
			if a.Arguments != nil {
				agent.Arguments.AddArgs(a.Arguments.GetAllArgs())
			}
		}
		addAll(req.Agent)
		if req.Arguments != nil {
			agent.Arguments.AddArgs(req.Arguments.GetAllArgs())
		}

		// init
		final := HandlerFunc(func(req *api.Request, res *api.Response) error {
			log.GetLogger(req.Context()).Infof("🔗 (final): %s\n", req.Name)
			return nil
		})
		chain := NewChain(sw.middlewares...).Then(agent, final)
		if err := chain.Serve(req, resp); err != nil {
			return err
		}

		if resp.Result == nil {
			// some thing went wrong
			return fmt.Errorf("Empty result running %q", agent.Name)
		}

		if resp.Result.State == api.StateTransfer {
			logger.Debugf("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
			req.Name = resp.Result.NextAgent
			continue
		}

		end := time.Now()
		logger.Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}

// // CallLLM calls the language model similar to Run but bypassing all the middlewares
// func (sw *Swarm) CallLLM(req *api.Request, resp *api.Response) error {
// 	if sw.User == nil || sw.Vars == nil {
// 		return api.NewInternalServerError("invalid config. user or vars not initialized")
// 	}
// 	if v, _ := sw.Vars.Global.Get("workspace"); v == "" {
// 		return api.NewInternalServerError("invalid config. user or vars not initialized")
// 	}
// 	if req.Parent != nil && req.Parent.Name == req.Name {
// 		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself not supported.", req.Name))
// 	}

// 	var ctx = req.Context()
// 	logger := log.GetLogger(ctx)

// 	for {
// 		start := time.Now()
// 		logger.Debugf("creating agent: %s %s\n", req.Name, start)

// 		// creator
// 		agent, err := sw.CreateAgent(ctx, req.Name)
// 		if err != nil {
// 			return err
// 		}

// 		// init
// 		final := HandlerFunc(func(req *api.Request, res *api.Response) error {
// 			log.GetLogger(req.Context()).Infof("🔗 (final): %s\n", req.Name)
// 			return nil
// 		})
// 		middlewares := []api.Middleware{
// 			InferenceMiddleware(sw),
// 		}
// 		chain := NewChain(middlewares...).Then(agent, final)
// 		if err := chain.Serve(req, resp); err != nil {
// 			return err
// 		}

// 		if resp.Result == nil {
// 			// some thing went wrong
// 			return fmt.Errorf("Empty result running %q", agent.Name)
// 		}

// 		if resp.Result.State == api.StateTransfer {
// 			logger.Debugf("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
// 			req.Name = resp.Result.NextAgent
// 			continue
// 		}

// 		end := time.Now()
// 		logger.Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
// 		return nil
// 	}
// }

// copy values from src to dst after calling @agent and applying template if required
func (sw *Swarm) mapAssign(ctx context.Context, agent *api.Agent, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if _, ok := dst[key]; ok && !override {
			continue
		}

		// // @agent value support
		// if v, ok := val.(string); ok {
		// 	if resolved, err := sw.resolveArgument(ctx, agent, v); err != nil {
		// 		return err
		// 	} else {
		// 		val = resolved
		// 	}
		// }

		// go template value support
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			if resolved, err := atm.ApplyTemplate(agent.Template, v, dst); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		dst[key] = val
	}
	return nil
}

// make a copy of golbal env
func (sw *Swarm) globalEnv() map[string]any {
	var env = make(map[string]any)
	sw.Vars.Global.Copy(env)
	return env
}

// // call agent if found. otherwise return s as is
// func (sw *Swarm) resolveArgument(ctx context.Context, agent *api.Agent, s string) (any, error) {
// 	if !conf.IsAgentTool(s) {
// 		return s, nil
// 	}
// 	out, err := sw.expand(ctx, agent, s)
// 	if err != nil {
// 		return nil, err
// 	}

// 	type ArgResult struct {
// 		Result string
// 		Error  string
// 	}

// 	var arg ArgResult
// 	if err := json.Unmarshal([]byte(out), &arg); err != nil {
// 		return nil, err
// 	}
// 	if arg.Error != "" {
// 		return nil, fmt.Errorf("failed resolve argument: %s", arg.Error)
// 	}
// 	return arg.Result, nil
// }

// expand s for agent/tool similar to $(cmdline...)
func (sw *Swarm) expandx(ctx context.Context, parent *api.Agent, s string) (string, error) {
	argv := shlex.Argv(s)
	data, err := sw.ExecCommand(ctx, parent, argv)
	if err != nil {
		return "", nil
	}
	return api.ToString(data), nil
}

func (sw *Swarm) Execv(ctx context.Context, arg0 string, argv []string) (*api.Result, error) {
	agent, err := sw.CreateAgent(ctx, arg0)
	if err != nil {
		return nil, err
	}
	data, err := sw.ExecCommand(ctx, agent, argv)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = ""
	}
	return api.ToResult(data), nil
}

func (sw *Swarm) Execm(ctx context.Context, arg0 string, args map[string]any) (*api.Result, error) {
	data, err := sw.RunAction(ctx, nil, arg0, args)
	if err != nil {
		return nil, err
	}
	if data == nil {
		data = ""
	}
	return api.ToResult(data), nil
}

func (sw *Swarm) ExecCommand(ctx context.Context, agent *api.Agent, argv []string) (any, error) {
	at, err := conf.ParseActionArgs(argv)
	if err != nil {
		return nil, err
	}
	if at == nil {
		return nil, fmt.Errorf("invalid agent tool command: %+v", argv)
	}
	id := api.KitName(at.Kit + ":" + at.Name).ID()

	return agent.Runner.Run(ctx, id, at.Arguments)
}

func (sw *Swarm) RunAction(ctx context.Context, parent *api.Agent, action string, args map[string]any) (any, error) {
	req := api.NewRequest(ctx, action, args)
	req.Agent = parent

	resp := &api.Response{}
	err := sw.Run(req, resp)
	if err != nil {
		return "", err
	}
	if resp.Result == nil {
		return "no output", nil
	}
	return resp.Result, nil
}

// inherit parent tools including embedded agents
// TODO cache
func (sw *Swarm) buildAgentToolMap(agent *api.Agent) map[string]*api.ToolFunc {
	toolMap := make(map[string]*api.ToolFunc)
	if agent == nil {
		return toolMap
	}
	// inherit tools of embedded agents
	for _, agent := range agent.Embed {
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}
	}
	// the active agent
	for _, v := range agent.Tools {
		toolMap[v.ID()] = v
	}
	return toolMap
}

func (sw *Swarm) callTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", tf.Kit, tf.Name, formatArgs(args))

	result, err := sw.dispatch(ctx, agent, tf, args)

	if err != nil {
		log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
	} else {
		log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
	}

	return result, err
}

func (sw *Swarm) callAgentType(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// agent tool
	if tf.Kit == api.ToolTypeAgent {
		return sw.RunAction(ctx, agent, tf.Agent, args)
	}

	// ai tool
	if tf.Kit == "ai" {
		return sw.callAIAgentTool(ctx, agent, tf, args)
	}
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (sw *Swarm) callAIAgentTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	aiKit := NewAIKit(sw, agent)
	return aiKit.Call(ctx, sw.Vars, "", tf, args)
}

func (sw *Swarm) dispatch(ctx context.Context, agent *api.Agent, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// agent tool
	if v.Type == api.ToolTypeAgent {
		out, err := sw.callAgentType(ctx, agent, v, args)
		if err != nil {
			return nil, err
		}
		return api.ToResult(out), nil
	}

	// custom kits
	kit, err := sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}

	env := &api.ToolEnv{
		Owner: agent.Owner,
		Agent: agent,
		FS:    NewToolFS(sw.Workspace),
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
	}
	return api.ToResult(out), nil
}
