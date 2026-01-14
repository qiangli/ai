package swarm

import (
	"context"
	"fmt"
	// "maps"
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
	// new env vars can only reference existing global vars
	// export all agent/embedded env
	var envs = make(map[string]any)
	addEnv := func(a *api.Agent) error {
		if a.Environment != nil {
			src := a.Environment.GetAllEnvs()
			return r.sw.mapAssign(ctx, a, envs, src, true)
		}
		return nil
	}
	if err := walkAgent(agent, addEnv); err != nil {
		return nil, err
	}
	// NOTE: args["environment"] ?

	// export envs
	vars.Global.AddEnvs(envs)
	// update - not used but for info?
	agent.Environment = vars.Global
	// agent.Environment.SetEnvs(envs)

	// *** args ***
	// global/agent envs
	// resolve agent args
	var agentArgs = make(map[string]any)
	if len(agent.Arguments) > 0 {
		if err := r.sw.mapAssign(ctx, agent, agentArgs, agent.Arguments, true); err != nil {
			return nil, err
		}
	}
	agent.Arguments = agentArgs
	// copy into input args if not exsiting
	// maps.Copy(args, agentArgs)

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
	// if model == nil {
	// 	provider, _ := api.GetStrProp("provider", args)
	// 	if provider != "" {
	// 		model = conf.DefaultModels[provider]
	// 	} else {
	// 		model = lookupModel(agent)
	// 	}
	// }
	var owner = r.sw.User.Email
	if v, found := args["model"]; found {
		switch vt := v.(type) {
		case *api.Model:
			model = vt
		case string:
			// set/level
			set, level := api.Setlevel(vt).Decode()
			// embeded/inherited
			if v := findModel(agent, set, level); v != nil {
				model = v
				break
			}
			// external
			v, err := conf.LoadModel(owner, set, level, r.sw.Assets)
			if err != nil {
				return nil, err
			}
			model = v
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
	// TODO merge: args["tools"]
	agent.Tools = list
	args["tools"] = list

	// update the property with the created agent object
	args["kit"] = "agent"
	args["name"] = agent.Name
	args["pack"] = agent.Pack
	args["agent"] = agent

	//
	for k, v := range agentArgs {
		if _, ok := args[k]; !ok {
			args[k] = v
		}
	}

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

func (r *AIKit) BuildPrompt(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (string, error) {
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
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

func (r *AIKit) BuildContext(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (any, error) {
	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
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
	var history = args.History()
	if !args.HasHistory() {
		if err := walkAgent(agent, addCtx); err != nil {
			return nil, err
		}

		for _, v := range contexts {
			// v = strings.TrimSpace(v)
			// // var list []*api.Message
			// if err := json.Unmarshal([]byte(v), &list); err != nil {
			// 	// best effort
			// 	list = []*api.Message{
			// 		{
			// 			Role:    api.RoleUser,
			// 			Content: v,
			// 		},
			// 	}
			// }
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
