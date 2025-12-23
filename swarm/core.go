package swarm

import (
	"context"
	"fmt"

	// "strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

type Swarm struct {
	// session id
	ID string

	User *api.User

	Secrets api.SecretStore

	Assets api.AssetManager

	Tools api.ToolSystem

	Adapters api.AdapterRegistry

	Blobs api.BlobStore

	// virtual system
	// Root      string
	OS        vos.System
	Workspace vfs.Workspace
	History   api.MemStore

	// runtime fields
	// internal
	middlewares []api.Middleware
	agentMaker  *AgentMaker

	//
	Vars *api.Vars
}

func (sw *Swarm) Init(rte *api.ActionRTEnv) error {
	if rte == nil {
		return fmt.Errorf("Action RT env required")
	}
	if rte.Base == "" {
		return fmt.Errorf("app base required")
	}
	// required
	if rte.Workspace == nil {
		return fmt.Errorf("app workspace not initialized")
	}
	if rte.User == nil {
		return fmt.Errorf("user not authenticated")
	}
	if rte.Secrets == nil {
		return fmt.Errorf("secret store not initialized")
	}
	if rte.OS == nil {
		return fmt.Errorf("execution env not avalable")
	}

	sw.InitChain()
	sw.Vars = api.NewVars()

	sw.Vars.RTE = rte

	maker := NewAgentMaker(sw)
	// TODO move to Vars?
	sw.agentMaker = maker

	root, err := maker.CreateFrom(context.TODO(), "root", resource.RootAgentData)
	if err != nil {
		return err
	}
	root.Runner = NewAgentToolRunner(sw, sw.User.Email, root)
	root.Shell = NewAgentScriptRunner(sw, root)
	root.Template = NewTemplate(sw, root)
	sw.Vars.RootAgent = root

	return nil
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
		ToolsMiddleware(sw),
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

// Create agent by name.
// Search assets and load the agent if found.
func (sw *Swarm) CreateAgent(ctx context.Context, name string) (*api.Agent, error) {
	if name == "" {
		// anonymous
		log.GetLogger(ctx).Debugf("agent not specified.\n")
	}
	agent, err := sw.agentMaker.Create(ctx, name)
	if err != nil {
		return nil, err
	}

	agent.Parent = sw.Vars.RootAgent

	return agent, nil
}

// Serve calls the language model with the messages list (after applying the system prompt).
// If the resulting AI Message contains tool_calls, the orchestrator will then call the tools.
// The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (sw *Swarm) Serve(req *api.Request, resp *api.Response) error {

	return sw.serve(sw.CreateAgent, req, resp)
}

func (sw *Swarm) serve(creator api.Creator, req *api.Request, resp *api.Response) error {
	if req.Agent == nil {
		req.Agent = sw.Vars.RootAgent
	}

	var ctx = req.Context()
	logger := log.GetLogger(ctx)
	for {
		start := time.Now()
		logger.Debugf("creating agent: %s %s\n", req.Name, start)

		// creator
		// TODO inherit runner
		agent, err := creator(ctx, req.Name)
		if err != nil {
			return err
		}
		//
		agent.Runner = NewAgentToolRunner(sw, sw.User.Email, agent)
		agent.Shell = NewAgentScriptRunner(sw, agent)
		agent.Template = NewTemplate(sw, agent)

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
				agent.Arguments.AddArgs(a.Arguments)
			}
		}
		addAll(req.Agent)
		if req.Arguments != nil {
			agent.Arguments.AddArgs(req.Arguments)
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
			if resp.Result.NextAgent == "self" {
				name := agent.Name
				v, err := sw.creatorFromScript(name, resp.Result.Value)
				if err != nil {
					return err
				}
				creator = v
				logger.Infof("Agent reload: %s => %s\n", req.Name, resp.Result.NextAgent)
			} else {
				logger.Infof("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
			}

			req.Name = resp.Result.NextAgent
			req.Agent = agent
			continue
		}

		end := time.Now()
		logger.Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}

// copy values from src to dst after calling @agent and applying template if required
// skip unless override is true
func (sw *Swarm) mapAssign(ctx context.Context, agent *api.Agent, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if _, ok := dst[key]; ok && !override {
			continue
		}

		// go template value support
		if atm.IsTemplate(val) {
			if resolved, err := atm.CheckApplyTemplate(agent.Template, val.(string), dst); err != nil {
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

func (sw *Swarm) globalAddEnvs(envs map[string]any) {
	sw.Vars.Global.AddEnvs(envs)
}

// expand s for agent/tool similar to $(cmdline...)
func (sw *Swarm) expandx(ctx context.Context, parent *api.Agent, s string) (string, error) {
	data, err := sw.exec(ctx, parent, s)
	if err != nil {
		return "", nil
	}
	return api.ToString(data), nil
}

func (sw *Swarm) Parse(ctx context.Context, input any) (api.ArgMap, error) {
	// parse special chars: - }}
	parsev := func(argv []string) (api.ArgMap, error) {
		if conf.IsAction(argv[0]) {
			cfg, err := GetInput(ctx, argv)
			if err != nil {
				return nil, err
			}
			// remove special trailing chars
			if cfg.Message != "" {
				argv = append(cfg.Args, "--stdin", cfg.Message)
			}
		}
		return conf.Parse(argv)
	}

	switch input := input.(type) {
	case string:
		argv := conf.Argv(input)
		return parsev(argv)
	case []string:
		return parsev(input)
	}
	return conf.Parse(input)
}

func (sw *Swarm) Format(ctx context.Context, argm map[string]any) (*api.Result, error) {
	format, _ := api.GetStrProp("format", argm)
	if format == "" {
		format = "markdown"
	}
	var v string
	var tpl = resource.FormatFile(format)
	atm.CheckApplyTemplate(sw.Vars.RootAgent.Template, tpl, argm)
	return &api.Result{
		Value: v,
	}, nil
}

func (sw *Swarm) Exec(ctx context.Context, input any) (*api.Result, error) {
	return sw.exec(ctx, sw.Vars.RootAgent, input)
}

func (sw *Swarm) exec(ctx context.Context, parent *api.Agent, input any) (*api.Result, error) {
	argm, err := conf.Parse(input)
	if err != nil {
		return nil, err
	}
	return sw.execm(ctx, parent, argm)
}

// default action runner
func (sw *Swarm) execm(ctx context.Context, parent *api.Agent, argm map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Debugf("argm: %+v\n", argm)

	am := api.ArgMap(argm)
	id := am.Kitname().ID()
	if id == "" {
		// required
		// kit is optional for system command
		return nil, fmt.Errorf("missing action id: %+v", argm)
	}
	v, err := parent.Runner.Run(ctx, id, argm)
	if err != nil {
		return nil, err
	}
	result := api.ToResult(v)
	return result, nil
}

func (sw *Swarm) creatorFromScript(name, script string) (api.Creator, error) {
	data, err := sw.LoadScript(script)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("agent name required")
	}
	pack, _ := api.Packname(name).Decode()
	v, err := sw.agentMaker.Creator(sw.agentMaker.Create, sw.User.Email, pack, []byte(data))
	if err != nil {
		return nil, err
	}
	return v, nil
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
		// in case nil is returned by the tools
		if result == nil {
			result = &api.Result{}
		}
		log.GetLogger(ctx).Infof("✔ %s (%s)\n", tf.ID(), head(result.String(), 180))
		log.GetLogger(ctx).Debugf("details:\n%s\n", result.String())
	}

	return result, err
}

func (sw *Swarm) dispatch(ctx context.Context, agent *api.Agent, v *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	// command
	if v.Type == api.ToolTypeBin {
		out, err := atm.ExecCommand(ctx, sw.OS, sw.Vars, v.Name, nil)
		if err != nil {
			return nil, err
		}
		return &api.Result{
			Value: out,
		}, nil
	}

	// ai
	if v.Type == api.ToolTypeAI {
		aiKit := NewAIKit(sw, agent)
		out, err := aiKit.Call(ctx, sw.Vars, v, args)
		if err != nil {
			return nil, err
		}
		return api.ToResult(out), nil
	}

	// agent tool
	if v.Type == api.ToolTypeAgent {
		aiKit := NewAIKit(sw, agent)
		args["agent"] = v.Name
		return aiKit.SpawnAgent(ctx, sw.Vars, "", args)
	}

	// misc kits
	kit, err := sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}

	env := &api.ToolEnv{
		Agent: agent,
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)

	if err != nil {
		return nil, err
	}
	return api.ToResult(out), nil
}

func (sw *Swarm) LoadScript(v string) (string, error) {
	return api.LoadURIContent(sw.Workspace, v)
}
