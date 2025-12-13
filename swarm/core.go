package swarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
	"github.com/qiangli/ai/swarm/llm"
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

	Adapters llm.AdapterRegistry

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
	// rte.Roots optional
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

	// // required by toolkit
	// sw.Vars.RTE = &api.ActionRTEnv{
	// 	// Root:      sw.Root,
	// 	User:      sw.User,
	// 	Secrets:   sw.Secrets,
	// 	Workspace: sw.Workspace,
	// 	OS:        sw.OS,
	// }

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
	// if sw.User == nil || sw.Vars == nil {
	// 	return api.NewInternalServerError("invalid config. user or vars not initialized")
	// }
	// if v, _ := sw.Vars.Global.Get("workspace"); v == "" {
	// 	return api.NewInternalServerError("invalid config. user or vars not initialized")
	// }
	// if req.Agent != nil && req.Agent.Name == req.Name {
	// 	return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself not supported.", req.Name))
	// }
	// if req.Agent == nil {
	// 	req.Agent = sw.Vars.RootAgent
	// }

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
		//
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
			req.Agent = agent
			continue
		}

		end := time.Now()
		logger.Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}

// copy values from src to dst after calling @agent and applying template if required
func (sw *Swarm) mapAssign(ctx context.Context, agent *api.Agent, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if _, ok := dst[key]; ok && !override {
			continue
		}

		// go template value support
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			if resolved, err := atm.CheckApplyTemplate(agent.Template, v, dst); err != nil {
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

// expand s for agent/tool similar to $(cmdline...)
func (sw *Swarm) expandx(ctx context.Context, parent *api.Agent, s string) (string, error) {
	data, err := sw.exec(ctx, parent, s)
	if err != nil {
		return "", nil
	}
	return api.ToString(data), nil
}

// // Convert arg string and run agent action
// func (sw *Swarm) Run(ctx context.Context, parent *api.Agent, args string) (*api.Result, error) {
// 	argm, err := sw.Parse(ctx, args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sw.Exec(ctx, argm)
// }

// // Convert arg string and run agent action
// func (sw *Swarm) Run(ctx context.Context, parent *api.Agent, args string) (*api.Result, error) {
// 	// argm, err := sw.Parse(ctx, args)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// am := api.ArgMap(argm)
// 	// kit := am.Kit()
// 	// name := am.Name()
// 	// if kit != string(api.ToolTypeAgent) {
// 	// 	return nil, fmt.Errorf("invalid agent: %v", name)
// 	// }
// 	// return sw.runm(ctx, parent, name, am)
// 	return sw.exec(ctx, parent, args)
// }

// // Convert arg array and run agent action
// func (sw *Swarm) Runv(ctx context.Context, parent *api.Agent, argv []string) (*api.Result, error) {
// 	argm, err := conf.ParseActionArgs(argv)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sw.Runm(ctx, parent, argm)
// }

// // Run agent action
// func (sw *Swarm) Runm(ctx context.Context, parent *api.Agent, argm map[string]any) (*api.Result, error) {
// 	am := api.ArgMap(argm)
// 	kit := am.Kit()
// 	name := am.Name()
// 	if kit != string(api.ToolTypeAgent) {
// 		return nil, fmt.Errorf("invalid agent: %v", name)
// 	}
// 	return sw.runm(ctx, parent, name, am)
// }

func (sw *Swarm) Parse(ctx context.Context, input any) (api.ArgMap, error) {
	// parse special chars: - }}
	parsev := func(argv []string) (api.ArgMap, error) {
		if conf.IsAction(argv[0]) {
			cfg, err := GetInput(ctx, argv)
			if err != nil {
				return nil, err
			}
			// remove special trailing chars
			argv = append(cfg.Args, "--stdin", cfg.Message)
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

// func (sw *Swarm) parses(ctx context.Context, args string) (api.ArgMap, error) {
// 	argv := conf.Argv(args)
// 	return sw.parsev(ctx, argv)
// }

// func (sw *Swarm) parsev(ctx context.Context, argv []string) (api.ArgMap, error) {
// 	var argm map[string]any

// 	if conf.IsAction(argv[0]) {
// 		cfg, err := GetInput(ctx, argv)
// 		if err != nil {
// 			return nil, err
// 		}

// 		// remove special trailing chars
// 		argv = cfg.Args
// 		v, err := conf.ParseActionArgs(argv)
// 		if err != nil {
// 			return nil, err
// 		}
// 		argm = v

// 		msg := argm["message"]
// 		if cfg.Message != "" {
// 			argm["message"] = Cat(msg.(string), cfg.Message, "\n###\n")
// 		}
// 	} else if conf.IsSlash(argv[0]) {
// 		// call local system command as tool:
// 		// sh:exec command
// 		argm = make(map[string]any)
// 		argm["kit"] = "sh"
// 		argm["name"] = "exec"
// 		argm["command"] = strings.Join(argv, " ")
// 	} else {
// 		argm = make(map[string]any)
// 		argm["message"] = strings.Join(argv, " ")
// 	}

// 	return sw.parsem(ctx, argm)
// }

// func (sw *Swarm) parsem(ctx context.Context, argm map[string]any) (api.ArgMap, error) {
// 	if len(argm) == 0 {
// 		return nil, fmt.Errorf("empty map")
// 	}
// 	log.GetLogger(ctx).Debugf("Parsem %+v\n", argm)

// 	a := api.ArgMap(argm)
// 	id := a.Kitname().ID()
// 	if id == "" {
// 		return nil, fmt.Errorf("missing action id: %+v", argm)
// 	}
// 	return a, nil
// }

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
	// switch input := input.(type) {
	// case string:
	// 	return sw.execs(ctx, parent, input)
	// case []string:
	// 	return sw.execv(ctx, parent, input)
	// case map[string]any:
	// 	return sw.execm(ctx, parent, input)
	// }
	// return nil, fmt.Errorf("not supported %t", input)
	argm, err := conf.Parse(input)
	if err != nil {
		return nil, err
	}
	return sw.execm(ctx, parent, argm)
}

// func (sw *Swarm) execs(ctx context.Context, parent *api.Agent, args string) (*api.Result, error) {
// 	argm, err := sw.parses(ctx, args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sw.execm(ctx, parent, argm)
// }

// func (sw *Swarm) execv(ctx context.Context, parent *api.Agent, argv []string) (*api.Result, error) {
// 	argm, err := sw.parsev(ctx, argv)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return sw.execm(ctx, parent, argm)
// }

func (sw *Swarm) execm(ctx context.Context, parent *api.Agent, argm map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Debugf("Execm parent: %+v args:%+v\n", parent, argm)

	if parent == nil {
		parent = sw.Vars.RootAgent
	}

	am := api.ArgMap(argm)
	id := am.Kitname().ID()
	if id == "" {
		return nil, fmt.Errorf("missing action id: %+v", argm)
	}
	kit := am.Kit()
	name := am.Name()

	var v any
	var err error
	switch kit {
	case "agent":
		//
		v, err = sw.runm(ctx, parent, name, argm)
	default:
		// all tools including sh:bash
		v, err = parent.Runner.Run(ctx, id, argm)
	}
	if err != nil {
		return nil, err
	}
	result := api.ToResult(v)
	return result, nil
}

// Run agent action. if custom config is detected. try load the agent from it.
func (sw *Swarm) runm(ctx context.Context, parent *api.Agent, name string, args map[string]any) (*api.Result, error) {
	var creator = sw.CreateAgent

	// load agent from content
	if s, ok := args["script"]; ok {
		data, err := sw.LoadScript(api.ToString(s))
		if err != nil {
			return nil, err
		}
		pack, _ := api.Packname(name).Decode()
		if v, err := sw.agentMaker.Creator(sw.agentMaker.Create, sw.User.Email, pack, []byte(data)); err == nil {
			creator = v
		}
	}
	return sw.runc(ctx, creator, parent, name, args)
}

func (sw *Swarm) runc(ctx context.Context, creator api.Creator, parent *api.Agent, name string, args map[string]any) (*api.Result, error) {
	req := api.NewRequest(ctx, name, args)
	req.Agent = parent

	resp := &api.Response{}
	err := sw.serve(creator, req, resp)
	if err != nil {
		return nil, err
	}
	if resp.Result == nil {
		return nil, fmt.Errorf("no output")
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
	if tf.Kit == string(api.ToolTypeAgent) {
		return sw.runm(ctx, agent, tf.Agent, args)
	}

	// // ai tool
	// if tf.Kit == "ai" {
	// 	return sw.callAITool(ctx, agent, tf, args)
	// }
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (sw *Swarm) callAIType(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	aiKit := NewAIKit(sw, agent)
	return aiKit.Call(ctx, sw.Vars, tf, args)
}

func (sw *Swarm) dispatch(ctx context.Context, agent *api.Agent, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// ai
	if v.Type == api.ToolTypeAI {
		out, err := sw.callAIType(ctx, agent, v, args)
		if err != nil {
			return nil, err
		}
		return api.ToResult(out), nil
	}

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
		Agent: agent,
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)

	if err != nil {
		return nil, err
	}
	return api.ToResult(out), nil
}

func (sw *Swarm) LoadScript(v string) (string, error) {
	return atm.LoadScript(sw.Workspace, v)
}

// func LoadScript(ws api.Workspace, v string) (string, error) {
// 	var script string

// 	if strings.HasPrefix(v, "data:") {
// 		// FIXME remove mime
// 		script = v[5:]
// 	} else {
// 		file := v
// 		data, err := ws.ReadFile(file, nil)
// 		if err != nil {
// 			return "", err
// 		}
// 		script = string(data)
// 	}

// 	return script, nil
// }
