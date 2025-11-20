package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	// "maps"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/u-root/u-root/pkg/shlex"

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
}

// https://pkg.go.dev/text/template
// https://masterminds.github.io/sprig/
func (sw *Swarm) InitTemplate(agent *api.Agent) *template.Template {
	var fm = sprig.FuncMap()
	// overridge sprig
	fm["user"] = func() *api.User {
		return sw.User
	}
	// OS
	getenv := func(key string) string {
		v, ok := sw.Vars.Global.Get(key)
		if !ok {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	fm["env"] = getenv
	fm["expandenv"] = func(s string) string {
		// bash name is leaked with os.Expand but ok.
		// bash is replaced with own that supports executing agent/tool
		return os.Expand(s, getenv)
	}
	// Network:
	fm["getHostByName"] = func() string {
		return "localhost"
	}

	// ai
	fm["ai"] = func(args ...string) string {
		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err.Error()
		}
		id := api.KitName(at.Name).ID()

		ctx := context.Background()
		data, err := agent.Runner.Run(ctx, id, at.Arguments)
		if err != nil {
			return err.Error()
		}
		result := api.ToResult(data)
		if err != nil {
			return err.Error()
		}

		return result.Value
	}

	return template.New("swarm").Funcs(fm)
}

func (sw *Swarm) InitChain() {

	// logging, analytics, and debugging.
	// prompts, tool selection, and output formatting.
	// retries, fallbacks, early termination.
	// rate limits, guardrails, pii detection.
	sw.middlewares = []api.Middleware{
		InitEnvMiddleware(sw),
		//
		TimeoutMiddleware(sw),
		LogMiddleware(sw),
		//
		ModelMiddleware(sw),
		ToolMiddleware(sw),
		//
		MemoryMiddleware(sw),
		InstructionMiddleware(sw),
		QueryMiddleware(sw),
		ContextMiddleware(sw),
		//
		AgentMiddleware(sw),
		//
		InferenceMiddleware(sw),
		// output
	}
}

func (sw *Swarm) NewChain(ctx context.Context, a *api.Agent) api.Handler {
	final := HandlerFunc(func(req *api.Request, res *api.Response) error {
		log.GetLogger(req.Context()).Infof("🔗 (final): %s\n", req.Name)
		return nil
	})

	chain := NewChain(sw.middlewares...).Then(a, final)
	return chain
}

func (sw *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	var name = req.Name
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
	agent.Template = sw.InitTemplate(agent)
	return agent, nil
}

// Run calls the language model with the messages list (after applying the system prompt).
// If the resulting AI Message contains tool_calls, the orchestrator will then call the tools.
// The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (sw *Swarm) Run(req *api.Request, resp *api.Response) error {
	if sw.User == nil || sw.Vars == nil {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}

	if req.Parent != nil && req.Parent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself not supported.", req.Name))
	}

	var ctx = req.Context()
	logger := log.GetLogger(ctx)

	for {
		start := time.Now()
		logger.Debugf("creating agent: %s %s\n", req.Name, start)

		// creator
		agent, err := sw.createAgent(ctx, req)
		if err != nil {
			return err
		}

		// init
		if err := sw.NewChain(ctx, agent).Serve(req, resp); err != nil {
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

// copy values from src to dst after calling @agent and applying template if required
func (sw *Swarm) mapAssign(agent *api.Agent, req *api.Request, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if !override {
			if _, ok := dst[key]; ok {
				continue
			}
		}
		// @agent value support
		if v, ok := val.(string); ok {
			if resolved, err := sw.resolveArgument(agent, req, v); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		// go template value support
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			if resolved, err := applyTemplate(agent.Template, v, dst); err != nil {
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

// call agent if found. otherwise return s as is
func (sw *Swarm) resolveArgument(agent *api.Agent, req *api.Request, s string) (any, error) {
	out, err := sw.resolveCommand(agent, req, s)
	if err != nil {
		return nil, err
	}

	type ArgResult struct {
		Result string
		Error  string
	}

	var arg ArgResult
	if err := json.Unmarshal([]byte(out), &arg); err != nil {
		return nil, err
	}
	if arg.Error != "" {
		return nil, fmt.Errorf("failed resolve argument: %s", arg.Error)
	}
	return arg.Result, nil
}

func (sw *Swarm) resolveCommand(parent *api.Agent, req *api.Request, s string) (string, error) {
	if !conf.IsAgentTool(s) {
		return s, nil
	}
	return sw.RunCommand(req.Context(), parent, s)
}

func (sw *Swarm) RunCommand(ctx context.Context, parent *api.Agent, s string) (string, error) {
	argv := shlex.Argv(s)
	at, err := conf.ParseActionArgs(argv)
	if err != nil {
		return "", err
	}
	if at == nil {
		return "", fmt.Errorf("invalid agent tool call: %v", s)
	}
	return sw.RunAction(ctx, parent, at.Name, at.ToMap())
}

func (sw *Swarm) RunAction(ctx context.Context, parent *api.Agent, action string, args map[string]any) (string, error) {
	req := api.NewRequest(ctx, action, args)
	req.Parent = parent

	resp := &api.Response{}
	err := sw.Run(req, resp)
	if err != nil {
		return "", err
	}
	if resp.Result == nil {
		return "no output", nil
	}
	return resp.Result.Value, nil
}

// func (sw *Swarm) callAgent(parent *api.Agent, name string, message string) (string, error) {
// 	args := make(map[string]any)
// 	args["message"] = message
// 	maps.Copy(args, parent.Parent.Arguments)

// 	req := api.NewRequest(ctx, name, args)
// 	req.Parent = parent
// 	req.Name = strings.TrimPrefix(name, "@")
// 	req.Name = strings.TrimPrefix(name, "agent:")
// 	// prepend additional instruction to user query
// 	// req.SetQuery(concat('\n', message, req.Query()))

// 	resp := &api.Response{}

// 	err := sw.Run(req, resp)
// 	if err != nil {
// 		return "", err
// 	}
// 	if resp.Result == nil {
// 		return "", nil
// 	}
// 	return resp.Result.Value, nil
// }

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
		return sw.callAgentTool(ctx, agent, tf, args)
	}

	// ai tool
	if tf.Kit == "ai" {
		return sw.callAIAgentTool(ctx, agent, tf, args)
	}
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (sw *Swarm) callAgentTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	req := api.NewRequest(ctx, tf.Agent, args)
	req.Parent = agent

	resp := &api.Response{}

	err := sw.Run(req, resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
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
		return ToResult(out), nil
	}

	// custom kits
	kit, err := sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}

	env := &api.ToolEnv{
		Owner: agent.Owner,
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
	}
	return ToResult(out), nil
}
