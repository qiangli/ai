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

// // copy global env only if not set
// func (r *AIKit) applyGlobal(args map[string]any) {
// 	envs := r.sw.globalEnv()
// 	for k, v := range envs {
// 		if _, ok := args[k]; !ok {
// 			args[k] = v
// 		}
// 	}
// }

// merge properties from agent and app config
// apply templates on env, inherit from embedded agents, export env
// apply templates on args
// resolve model
// resolve tools, inherit from embedded agents
//
// context/instruction/message are not resolved - each is done separately as needed
func (r *AIKit) createAgent(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Agent, error) {
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

	// TODO investigate.
	// config file is stale in the bash script. Need to re-read for new config files
	//
	// var ac *api.AppConfig
	// cfg := args["config"]
	// if v, ok := cfg.(*api.AppConfig); ok {
	// 	ac = v
	// } else {
	// 	v, err := r.ReadAgentConfig(ctx, vars, tf, args)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	ac = v
	// }
	// ac, err := r.ReadAgentConfig(ctx, vars, tf, args)
	// if err != nil {
	// 	return nil, err
	// }

	// config
	var cfg []byte
	if v, found := args["script"]; found {
		if uri, ok := v.(string); ok {
			if strings.HasSuffix(uri, ".yaml") || strings.HasSuffix(uri, ".yml") {
				data, err := r.sw.LoadScript(uri)
				if err != nil {
					return nil, err
				}
				cfg = []byte(data)
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
	var envs = make(map[string]any)
	maps.Copy(envs, r.sw.globalEnv())

	// //
	// if agent.Environment != nil {
	// 	r.sw.mapAssign(ctx, agent, envs, agent.Environment.GetAllEnvs(), true)
	// 	r.sw.globalAddEnvs(envs)
	// }

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

	addAll(agent)
	r.sw.globalAddEnvs(envs)
	agent.Environment.SetEnvs(envs)

	// args
	// global/agent envs
	// agent args
	// local arg takes precedence, skip if it already exists
	if agent.Arguments != nil {
		if err := r.sw.mapAssign(ctx, agent, args, agent.Arguments, false); err != nil {
			return nil, err
		}
	}
	// r.applyGlobal(args)
	agent.Arguments = args

	// *** model ***
	var model = agent.Model

	if model == nil {
		provider, _ := api.GetStrProp("provider", args)
		if provider == "" {
			return nil, fmt.Errorf("model missing. provider is required")
		}
		model = conf.DefaultModels[provider]
	}
	args["model"] = model

	// *** tools ***
	// inherit tools of embeded agents
	// deduplicate/merge all tools including the current agent
	// child tools take precedence.
	var list = agent.Tools

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
		agent.Tools = list
	}
	args["tools"] = list

	// update the property with the created agent object
	args["kit"] = "agent"
	args["name"] = agent.Name
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
			v, err := atm.CheckApplyTemplate(agent.Template, msg, args)
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
		content, err := atm.CheckApplyTemplate(a.Template, in, args)
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

// func (r *AIKit) contextMemOption(argm api.ArgMap) *api.MemOption {
// 	return &api.MemOption{
// 		MaxHistory: argm.GetInt("max_history"),
// 		MaxSpan:    argm.GetInt("max_span"),
// 		Offset:     argm.GetInt("offset"),
// 		Roles:      argm.GetStringSlice("roles"),
// 	}

// }

func (r *AIKit) BuildContext(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	var contexts []string
	add := func(a *api.Agent, in string) error {
		// content, err := atm.CheckApplyTemplate(agent.Template, in, args)
		// var data = args
		// if len(a.Arguments) > 0 {
		// data := make(map[string]any)
		// maps.Copy(data, args)
		// 	maps.Copy(data, a.Arguments)
		// }
		content, err := atm.CheckApplyTemplate(a.Template, in, args)
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
		// inherit
		// r.applyGlobal(args)

		if err := addAll(agent); err != nil {
			return "", err
		}

		for _, v := range contexts {
			v = strings.TrimSpace(v)
			var list []*api.Message
			if err := json.Unmarshal([]byte(v), &list); err != nil {
				//return nil, fmt.Errorf("failed to resolve context: %v", err)
				// best effort?
				continue
			}
			history = append(history, list...)
		}

		// var c = agent.Context

		// if c != "" {
		// 	v, err := atm.CheckApplyTemplate(agent.Template, c, args)
		// 	if err != nil {
		// 		return nil, err
		// 	}

		// 	history = append(history, &api.Message{
		// 		Content: v,
		// 		Role:    api.RoleSystem,
		// 	})

		// 	// } else {
		// 	// 	// load defaults
		// 	// 	if v, err := r.sw.History.Load(r.contextMemOption(args)); err != nil {
		// 	// 		return nil, err
		// 	// 	} else {
		// 	// 		history = v
		// 	// 	}
		// }
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

// func (r *AIKit) BuildModel(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
// 	var agent = r.agent
// 	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
// 		agent = v
// 	}

// 	var model = agent.Model

// 	if model == nil {
// 		provider := args.GetString("provider")
// 		if provider == "" {
// 			return nil, fmt.Errorf("model missing. provider is required")
// 		}
// 		model = conf.DefaultModels[provider]
// 	}

// 	args["model"] = model
// 	return model, nil
// }

// func (r *AIKit) BuildTools(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
// 	var agent = r.agent
// 	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
// 		agent = v
// 	}

// 	var list []*api.ToolFunc
// 	// inherit tools of embeded agents
// 	// deduplicate/merge all tools including the current agent
// 	// child tools take precedence.
// 	if agent.Embed != nil {
// 		var tools = make(map[string]*api.ToolFunc)

// 		var addAll func(*api.Agent) error
// 		addAll = func(a *api.Agent) error {
// 			for _, v := range a.Embed {
// 				if err := addAll(v); err != nil {
// 					return err
// 				}
// 			}
// 			for _, v := range a.Tools {
// 				tools[v.ID()] = v
// 			}
// 			return nil
// 		}

// 		addAll(agent)

// 		for _, v := range tools {
// 			list = append(list, v)
// 		}
// 		args["tools"] = list
// 	}

// 	return list, nil
// }
