package swarm

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
)

func CreateAgent(ctx context.Context, vars *api.Vars, parent *api.Agent, packname api.Packname, config []byte) (*api.Agent, error) {
	var loader = NewConfigLoader(vars)

	if config != nil {
		// load data
		if err := loader.LoadContent(string(config)); err != nil {
			return nil, err
		}
	}

	agent, err := loader.Create(ctx, packname)
	if err != nil {
		return nil, err
	}

	// init setup
	// TODO optimize
	// embeded
	add := func(p, a *api.Agent) {
		a.Parent = p
		a.Runner = NewAgentToolRunner(vars, a)
		a.Shell = NewAgentScriptRunner(vars, a)
		a.Template = atm.NewTemplate(vars, a)
	}

	var addAll func(*api.Agent, *api.Agent)
	addAll = func(p, a *api.Agent) {
		for _, v := range a.Embed {
			addAll(p, v)
		}
		add(p, a)
	}

	addAll(parent, agent)

	return agent, nil
}

// copy values from src to dst after applying templates if requested
// skip unless override is true
// var in src template can reference global env
func mapAssign(_ context.Context, global *api.Environment, agent *api.Agent, dst, src map[string]any, override bool) error {
	if len(src) == 0 {
		return nil
	}
	var data = make(map[string]any)
	// sw.Vars.Global.GetAllEnvs()
	maps.Copy(data, global.GetAllEnvs())
	for key, val := range src {
		if _, ok := dst[key]; ok && !override {
			continue
		}
		// go template value support
		if api.IsTemplate(val) {
			maps.Copy(data, dst)
			if resolved, err := atm.CheckApplyTemplate(agent.Template, val.(string), data); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		dst[key] = val
	}
	return nil
}

// apply templates on env, inherit from embedded agents, export env
// apply templates on args
// resolve model, inherit from embedded agents or default to root agent.
// resolve tools, inherit from embedded agents
//
// context/instruction/message are not resolved - each is done separately as needed
// var precedence:
// global env
// app/agent env (merged and exported)
// app/agent args/agent parameters
// args
func (r *AIKit) createAgent(ctx context.Context, vars *api.Vars, parent *api.Agent, _ *api.ToolFunc, args map[string]any) (*api.Agent, error) {
	var name string
	if v, found := args["agent"].(*api.Agent); found {
		return v, nil
	}
	if v, found := args["agent"].(string); found {
		name = v
	}

	// fall back to kit:name
	if name == "" {
		kn := r.kitname(args)
		kit, v := kn.Decode()
		if kit != "agent" {
			return nil, fmt.Errorf("missing agent name")
		}
		name = v
	}

	// config
	loadScript := func(v string) (string, error) {
		return api.LoadURIContent(vars.Workspace, v)
	}

	var cfg []byte
	if v, found := args["script"]; found {
		if uri, ok := v.(string); ok {
			if strings.HasSuffix(uri, ".yaml") || strings.HasSuffix(uri, ".yml") {
				data, err := loadScript(uri)
				if err != nil {
					return nil, err
				}
				cfg = []byte("data:," + data)
			}
		}
	}

	//
	if parent == nil {
		parent = r.vars.RootAgent
	}

	pn := api.Packname(name)
	agent, err := CreateAgent(ctx, r.vars, parent, pn, cfg)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, fmt.Errorf("Failed to create agent: %s", pn)
	}

	// *** envs ***
	// export envs to global
	// new env vars can only reference existing global vars
	// export all agent/embedded env
	var envs = make(map[string]any)
	addEnv := func(a *api.Agent) error {
		if a.Environment != nil {
			src := a.Environment.GetAllEnvs()
			return mapAssign(ctx, vars.Global, a, envs, src, true)
		}
		return nil
	}
	if err := walkAgent(agent, addEnv); err != nil {
		return nil, err
	}
	// merge cmd line envs map
	if v, found := args["environment"]; found {
		v, err := api.ToMap(v)
		if err != nil {
			return nil, err
		}
		maps.Copy(envs, v)
	}

	// export envs
	vars.Global.AddEnvs(envs)
	// pointing to global env
	agent.Environment = vars.Global

	// *** args ***
	// resolve agent args
	var agentArgs = make(map[string]any)
	if len(agent.Arguments) > 0 {
		if err := mapAssign(ctx, vars.Global, agent, agentArgs, agent.Arguments, true); err != nil {
			return nil, err
		}
	}
	agent.Arguments = agentArgs

	// *** model ***
	// inherit from parent
	var lookupModel func(*api.Agent) *api.Model
	lookupModel = func(a *api.Agent) *api.Model {
		if a == nil {
			return nil
		}
		if model := a.Model; model != nil {
			return model
		}
		return lookupModel(a.Parent)
	}

	if agent.Model == nil {
		model := lookupModel(agent)
		if model == nil {
			model = r.vars.RootAgent.Model
		}
		agent.Model = model
	}

	// *** tools ***
	// inherit tools from embeded agents
	// deduplicate/merge all tools including the current agent
	// child tools take precedence.
	var list []*api.ToolFunc
	if len(agent.Embed) > 0 {
		var tools = make(map[string]*api.ToolFunc)
		addTool := func(a *api.Agent) error {
			for _, v := range a.Tools {
				tools[v.ID()] = v
			}
			return nil
		}

		if err := walkAgent(agent, addTool); err != nil {
			return nil, err
		}

		for _, v := range tools {
			list = append(list, v)
		}
	} else {
		list = agent.Tools
	}

	agent.Tools = list

	// NOTE: cmdline args take precedence over parameter defaults and agent arguments.
	// defaults from parameters
	if len(agent.Parameters) > 0 {
		// maps.Copy(args, agent.Parameters.Defaults())
		for k, v := range agent.Parameters.Defaults() {
			if _, ok := args[k]; !ok {
				args[k] = v
			}
		}
	}
	// agent arguments
	for k, v := range agentArgs {
		if _, ok := args[k]; !ok {
			args[k] = v
		}
	}

	// update the property with the created agent object
	args["kit"] = "agent"
	args["name"] = agent.Name
	args["pack"] = agent.Pack
	args["agent"] = agent

	return agent, nil
}

func (r *AIKit) BuildQuery(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (any, error) {
	var agent = parent
	if v, err := r.checkAndCreate(ctx, vars, parent, tf, args); err == nil {
		agent = v
	}

	// convert user message into query if not set
	var query = args.Query()
	if !args.HasQuery() {
		msg := agent.Message
		if msg != "" {
			data := atm.BuildEffectiveArgs(vars, agent, args)
			v, err := atm.CheckApplyTemplate(agent.Template, msg, data)
			if err != nil {
				return "", err
			}
			msg = v
		}
		userMsg := args.Message()

		//
		query = api.Cat(msg, userMsg, "\n")
		args.SetQuery(query)
	}

	return query, nil
}

func (r *AIKit) BuildPrompt(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (any, error) {
	var agent = parent
	if v, err := r.checkAndCreate(ctx, vars, parent, tf, args); err == nil {
		agent = v
	}

	var instructions []string
	addInst := func(a *api.Agent) error {
		in := a.Instruction
		if in != "" {
			data := atm.BuildEffectiveArgs(vars, a, args)
			content, err := atm.CheckApplyTemplate(a.Template, in, data)
			if err != nil {
				return err
			}
			instructions = append(instructions, content)
		}
		return nil
	}

	// system role instructions
	// inherit from embeds
	var prompt = args.Prompt()
	if !args.HasPrompt() {
		if err := walkAgent(agent, addInst); err != nil {
			return "", err
		}
		prompt = strings.Join(instructions, "\n")
		args.SetPrompt(prompt)
	}

	return prompt, nil
}

func (r *AIKit) BuildContext(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (any, error) {
	var agent = parent
	if v, err := r.checkAndCreate(ctx, vars, parent, tf, args); err == nil {
		agent = v
	}

	var contexts []string
	addCtx := func(a *api.Agent) error {
		in := a.Context
		if in != "" {
			data := atm.BuildEffectiveArgs(vars, a, args)
			content, err := atm.CheckApplyTemplate(a.Template, in, data)
			if err != nil {
				return err
			}
			contexts = append(contexts, content)
		}
		return nil
	}

	// add context as user role message
	// inherit from embeds
	var history = args.History()
	if !args.HasHistory() {
		if err := walkAgent(agent, addCtx); err != nil {
			return nil, err
		}

		for _, v := range contexts {
			msg := &api.Message{
				Role:    api.RoleUser,
				Content: v,
			}
			history = append(history, msg)
		}
	}

	if history == nil {
		history = []*api.Message{}
	}
	if len(history) > 0 {
		args.SetHistory(history)
	} else {
		args.DeleteHitory()
	}
	return history, nil
}

func walkAgent(root *api.Agent, visit func(*api.Agent) error) error {
	seen := make(map[api.Packname]bool)
	var dfs func(*api.Agent) error
	dfs = func(n *api.Agent) error {
		id := api.NewPackname(n.Pack, n.Name)
		if seen[id] {
			return nil
		}
		seen[id] = true

		for _, c := range n.Embed {
			if err := dfs(c); err != nil {
				return err
			}
		}

		if err := visit(n); err != nil {
			return err
		}

		return nil
	}
	return dfs(root)
}
