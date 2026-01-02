package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
)

// merge properties from agent and app config
// apply templates on env, inherit from embedded agents, export env
// apply templates on args
// resolve model
// resolve tools, inherit from embedded agents
//
// context/instruction/message are not resolved - each is done separately as needed
func (r *AIKit) createAgent(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Agent, error) {
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
	var cfg []byte
	if v, found := args["script"]; found {
		if uri, ok := v.(string); ok {
			if strings.HasSuffix(uri, ".yaml") || strings.HasSuffix(uri, ".yml") {
				data, err := r.sw.LoadScript(uri)
				if err != nil {
					return nil, err
				}
				cfg = []byte("data:," + data)
			}
		}
	}

	agent, err := r.sw.CreateAgent(ctx, r.sw.Vars.RootAgent, api.Packname(name), cfg)
	if err != nil {
		return nil, err
	}

	// *** envs ***
	// export envs to global
	// new vars can only reference existing global vars
	// export all agent/embedded env
	var envs = make(map[string]any)
	maps.Copy(envs, vars.Global.GetAllEnvs())

	// inherit envs of embeded agents
	add := func(e *api.Environment) error {
		return r.sw.mapAssign(ctx, agent, envs, e.GetAllEnvs(), true)
	}

	var addAll func(*api.Agent) error
	addAll = func(a *api.Agent) error {
		for _, v := range a.Embed {
			if err := addAll(v); err != nil {
				return err
			}
		}
		if a.Environment != nil {
			if err := add(a.Environment); err != nil {
				return err
			}
		}
		return nil
	}

	if err := addAll(agent); err != nil {
		return nil, err
	}
	vars.Global.AddEnvs(envs)
	agent.Environment.SetEnvs(envs)

	// args
	// global/agent envs
	// agent args
	// local arg takes precedence, skip if it already exists
	if len(agent.Arguments) > 0 {
		if err := r.sw.mapAssign(ctx, agent, args, agent.Arguments, false); err != nil {
			return nil, err
		}
	}
	agent.Arguments = args

	// *** model ***
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

	// resolve model or inherit
	var model = agent.Model
	if model == nil {
		provider, _ := api.GetStrProp("provider", args)
		if provider != "" {
			model = conf.DefaultModels[provider]
		} else {
			model = lookupModel(agent)
		}
	}
	// default/any
	if model == nil {
		model = r.sw.Vars.RootAgent.Model
	}
	args["model"] = model

	// *** tools ***
	// inherit tools of embeded agents
	// deduplicate/merge all tools including the current agent
	// child tools take precedence.
	var list []*api.ToolFunc
	if len(agent.Embed) > 0 {
		var tools = make(map[string]*api.ToolFunc)

		var addAll func(*api.Agent) error
		addAll = func(a *api.Agent) error {
			for _, v := range a.Embed {
				if err := addAll(v); err != nil {
					return err
				}
			}
			for _, v := range a.Tools {
				tools[v.ID()] = v
			}
			return nil
		}

		addAll(agent)

		for _, v := range tools {
			list = append(list, v)
		}
	} else {
		list = agent.Tools
	}
	agent.Tools = list
	args["tools"] = list

	// update the property with the created agent object
	args["kit"] = "agent"
	args["name"] = agent.Name
	args["pack"] = agent.Pack
	args["agent"] = agent

	return agent, nil
}

func (r *AIKit) BuildQuery(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (string, error) {
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	// convert user message into query if not set
	var query = args.Query()
	if !args.HasQuery() {
		// r.applyGlobal(args)

		msg := agent.Message
		if msg != "" {
			// var data = make(map[string]any)
			// maps.Copy(data, agent.Arguments)
			// maps.Copy(data, vars.Global.GetAllEnvs())
			// maps.Copy(data, args)
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

func (r *AIKit) BuildPrompt(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (string, error) {
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	var instructions []string

	add := func(a *api.Agent, in string) error {
		// content, err := atm.CheckApplyTemplate(agent.Template, in, args)
		// var data = make(map[string]any)
		// maps.Copy(data, vars.Global.GetAllEnvs())
		// maps.Copy(data, a.Arguments)
		// maps.Copy(data, args)
		data := atm.BuildEffectiveArgs(vars, a, args)
		content, err := atm.CheckApplyTemplate(a.Template, in, data)
		if err != nil {
			return err
		}

		// update instruction
		instructions = append(instructions, content)
		return nil
	}

	var addAll func(*api.Agent) error

	// inherit embedded agent instructions
	// merge all including the current agent
	addAll = func(a *api.Agent) error {
		for _, v := range a.Embed {
			if err := addAll(v); err != nil {
				return err
			}
		}
		in := a.Instruction
		if in != "" {
			if err := add(a, in); err != nil {
				return err
			}
		}
		return nil
	}

	// system role instructions
	var prompt = args.Prompt()
	if !args.HasPrompt() {
		// r.applyGlobal(args)

		if err := addAll(agent); err != nil {
			return "", err
		}

		prompt = strings.Join(instructions, "\n")
		args.SetPrompt(prompt)
	}

	return prompt, nil
}

func (r *AIKit) BuildContext(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	var contexts []string
	add := func(a *api.Agent, input string) error {
		// var data = make(map[string]any)
		// maps.Copy(data, a.Arguments)
		// maps.Copy(data, vars.Global.GetAllEnvs())
		// maps.Copy(data, args)
		data := atm.BuildEffectiveArgs(vars, a, args)
		content, err := atm.CheckApplyTemplate(a.Template, input, data)
		if err != nil {
			return err
		}

		contexts = append(contexts, content)
		return nil
	}
	var addAll func(*api.Agent) error

	// inherit embedded agent contexts
	// merge all including the current agent
	addAll = func(a *api.Agent) error {
		for _, v := range a.Embed {
			if err := addAll(v); err != nil {
				return err
			}
		}
		in := a.Context
		if in != "" {
			if err := add(a, in); err != nil {
				return err
			}
		}
		return nil
	}

	var history = args.History()

	// add context as system role message
	if !args.HasHistory() {
		if err := addAll(agent); err != nil {
			return "", err
		}

		for _, v := range contexts {
			v = strings.TrimSpace(v)
			var list []*api.Message
			if err := json.Unmarshal([]byte(v), &list); err != nil {
				// best effort
				continue
			}
			history = append(history, list...)
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
