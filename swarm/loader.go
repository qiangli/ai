package swarm

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
)

type ConfigLoader struct {
	data []byte
	rte  *api.ActionRTEnv
}

func NewConfigLoader(rte *api.ActionRTEnv) *ConfigLoader {
	return &ConfigLoader{
		rte: rte,
	}
}

func (r *ConfigLoader) LoadContent(s string) ([]byte, error) {
	v, err := api.LoadURIContent(r.rte.Workspace, s)
	if err != nil {
		return nil, err
	}
	r.data = []byte(v)
	return r.data, nil
}

// load config for agent/tool
func (r *ConfigLoader) LoadConfig(s string) (*api.AppConfig, error) {
	kn := api.Kitname(s)
	kit, _ := kn.Decode()
	if kit == string(api.ToolTypeAgent) {
		return r.LoadAgentConfig(api.Packname(s))
	} else {
		return r.LoadToolConfig(kn)
	}
}

func (r *ConfigLoader) LoadAgentConfig(pn api.Packname) (*api.AppConfig, error) {
	pack, sub := pn.Decode()

	// equqal or the primary agent sub == ""/sub == pack
	equal := func(s string) bool {
		return s == sub || sub == "" && s == pack
	}

	if r.data != nil {
		ac, err := conf.LoadAgentsData([][]byte{r.data})
		if err != nil {
			return nil, err
		}

		if ac.Pack != pack {
			return nil, fmt.Errorf("wrong pack: %s config: %s", pack, ac.Pack)
		}

		for _, v := range ac.Agents {
			if equal(v.Name) {
				return ac, nil
			}
		}
		// continue to find
	}

	ac, err := r.rte.Assets.FindAgent(r.rte.User.Email, pack)
	if err != nil {
		return nil, err
	}
	for _, v := range ac.Agents {
		if equal(v.Name) {
			return ac, nil
		}
	}
	return nil, fmt.Errorf("config not found for: %s", pn)

}

func (r *ConfigLoader) LoadToolConfig(kn api.Kitname) (*api.AppConfig, error) {
	kit, name := kn.Decode()
	// equal or default if name == ""
	equal := func(n string) bool {
		return n == name || name == "" && n == kit
	}

	if r.data != nil {
		tc, err := conf.LoadToolData([][]byte{r.data})
		if err != nil {
			return nil, err
		}
		if tc.Kit != kit {
			return nil, fmt.Errorf("wrong kit: %s config: %s", kit, tc.Kit)
		}
		for _, v := range tc.Tools {
			if equal(v.Name) {
				return tc, nil
			}
		}
		// continue to find
	}

	tc, err := r.rte.Assets.FindToolkit(r.rte.User.Email, kit)
	if err != nil {
		return nil, err
	}
	for _, v := range tc.Tools {
		if equal(v.Name) {
			return tc, nil
		}
	}
	return nil, fmt.Errorf("could not find config for: %s", name)
}

func (r *ConfigLoader) CreateTool(tid string) (*api.ToolFunc, error) {
	kn := api.Kitname(tid)
	kit, name := kn.Decode()
	// /agent:
	if kit == string(api.ToolTypeAgent) {
		pn := api.Packname(name)
		ac, err := r.LoadAgentConfig(pn)
		if err != nil {
			return nil, err
		}
		_, sub := pn.Decode()
		v, err := conf.LoadAgentTool(ac, sub)
		return v, err
	} else {
		// /kit:tool
		tc, err := r.LoadToolConfig(kn)
		if err != nil {
			return nil, err
		}

		tools, err := conf.LoadTools(tc, r.rte.User.Email, r.rte.Secrets)
		if err != nil {
			return nil, err
		}
		equal := func(n string) bool {
			return n == name || name == "" && n == kit
		}
		for _, v := range tools {
			if equal(v.Name) {
				return v, nil
			}
		}
	}
	return nil, nil
}

func (r *ConfigLoader) NewAgent(c *api.AgentConfig, pn api.Packname) (*api.Agent, error) {
	// ac, err := r.LoadConfig(name)
	// if err != nil {
	// 	return nil, err
	// }
	// pn := api.Packname(name)

	// pack, _ := pn.Decode()
	// if ac.Pack != pack {
	// 	return nil, fmt.Errorf("wrong pack: %s config: %s.", pack, ac.Pack)
	// }

	// // // find agent config
	// // var c *api.AgentConfig
	// // for _, v := range ac.Agents {
	// // 	if pn.Equal(v.Name) {
	// // 		c = v
	// // 		break
	// // 	}
	// // }
	// if c == nil {
	// 	return nil, fmt.Errorf("agent %q not in config: %s", pn.Encode(), ac.Pack)
	// }

	ac := c.Config

	// new agent
	var agent = api.Agent{
		Adapter: c.Adapter,
		//
		Name:        c.Name,
		Display:     c.Display,
		Description: c.Description,
		//
		Instruction: c.Instruction,
		Context:     c.Context,
		Message:     c.Message,
		//
		Arguments: api.NewArguments(),
	}
	//
	args := make(map[string]any)
	maps.Copy(args, ac.ToMap())
	maps.Copy(args, c.ToMap())

	maxTurns := nzl(c.MaxTurns, ac.MaxTurns, defaultMaxTurns)
	maxTime := nzl(c.MaxTime, ac.MaxTime, defaultMaxTime)
	// hard limit
	maxTurns = min(maxTurns, maxTurnsLimit)
	maxTime = min(maxTime, maxTimeLimit)

	args["max_turns"] = maxTurns
	args["max_time"] = maxTime

	maxHistory := nzl(c.MaxHistory, ac.MaxHistory, defaultMaxHistory)
	maxSpan := nzl(c.MaxSpan, ac.MaxSpan, defaultMaxSpan)
	args["max_history"] = maxHistory
	args["max_span"] = maxSpan

	// log
	args["log_level"] = nvl(c.LogLevel, ac.LogLevel)

	agent.Arguments.SetArgs(args)

	// merge global vars
	agent.Environment = api.NewEnvironment()
	agent.Environment.AddEnvs(ac.Environment)
	agent.Environment.AddEnvs(c.Environment)

	// llm model set[/level]
	// @model support
	// flow does not require a model
	model := strings.TrimSpace(nvl(c.Model, ac.Model))
	if model != "" && agent.Flow == nil {
		if strings.HasPrefix(model, "@") {
			// defer model provider resolution
			agent.Model = &api.Model{
				Model: model,
			}
		} else {
			set, level := resolveModelLevel(model)
			// local
			if set == ac.Set {
				for k, v := range ac.Models {
					if k == level {
						agent.Model = &api.Model{
							Model: v.Model,
							//
							Provider: nvl(v.Provider, ac.Provider),
							BaseUrl:  nvl(v.BaseUrl, ac.BaseUrl),
							ApiKey:   nvl(v.ApiKey, ac.ApiKey),
						}
					}
				}
			}
			// load external model if not defined locally
			if agent.Model == nil {
				if v, err := conf.LoadModel(r.rte.User.Email, set, level, r.rte.Assets); err != nil {
					return nil, fmt.Errorf("failed to load model: %s %v", model, err)

				} else {
					agent.Model = v
				}
			}
		}
	}

	// tools
	funcMap := make(map[string]*api.ToolFunc)
	// kit:*
	for _, v := range c.Functions {
		var tools []*api.ToolFunc
		// local scope
		if v, err := conf.LoadLocalToolFunc(ac, r.rte.User.Email, v, r.rte.Secrets, r.rte.Assets); err != nil {
			return nil, err
		} else {
			tools = v
		}
		// load external kit if not defined locally
		if tools == nil {
			if v, err := conf.LoadToolFunc(r.rte.User.Email, v, r.rte.Secrets, r.rte.Assets); err != nil {
				return nil, err
			} else {
				tools = v
			}
		}

		for _, fn := range tools {
			id := fn.ID()
			if id == "" {
				return nil, fmt.Errorf("agent tool ID is empty: %s", c.Name)
			}
			funcMap[id] = fn
		}
	}
	var funcs []*api.ToolFunc
	for _, v := range funcMap {
		funcs = append(funcs, v)
	}
	agent.Tools = funcs

	// flow
	if c.Flow != nil {
		var actionMap = make(map[string]*api.Action)
		for _, v := range agent.Tools {
			actionMap[v.Kit+":"+v.Name] = api.NewAction(
				v.ID(),
				v.Name,
				v.Arguments,
			)
		}
		flow := &api.Flow{
			Type:   c.Flow.Type,
			Script: c.Flow.Script,
		}

		for _, v := range c.Flow.Actions {
			a, ok := actionMap[v]
			if !ok {
				return nil, fmt.Errorf("action missing: %s %s", agent.Name, v)
			}
			flow.Actions = append(flow.Actions, a)
		}
		agent.Flow = flow
	}

	return &agent, nil
}

// create agent (class) from config
func (r *ConfigLoader) Create(ctx context.Context, ac *api.AppConfig, packname api.Packname) (*api.Agent, error) {

	findConfig := func(ac *api.AppConfig, pn api.Packname) (*api.AgentConfig, error) {
		for _, a := range ac.Agents {
			if pn.Equal(a.Name) {
				return a, nil
			}
		}
		return nil, fmt.Errorf("no such agent: %s", pn)
	}

	// create the agent
	// agent: pack/sub
	// var user = ap.sw.User.Email
	pack, sub := packname.Decode()
	// todo have decode return pack
	if sub == "" {
		sub = pack
	}

	//
	if pack == "" {
		return nil, fmt.Errorf("missing agent pack")
	}

	// cached agent
	key := AgentCacheKey{
		User: r.rte.User.Email,
		Pack: pack,
		Sub:  sub,
	}
	// return a cloned copy if found
	if v, ok := agentCache.Get(key); ok {
		// log.GetLogger(ctx).Debugf("Using cached agent: %+v", key)
		return v.Clone(), nil
	}

	// var ent *api.Record
	// if v, err := r.sw.Assets.SearchAgent(ap.sw.User.Email, pack); err != nil {
	// 	return nil, err
	// } else {
	// 	ent = v
	// }
	// // invalid agent
	// if ent == nil && pack != "" {
	// 	return nil, fmt.Errorf("agent not found: %s", pack)
	// }

	// // access to models/tools is implicitly granted if user has permission to run the agent
	// // agent config
	// ac, err := ap.getAgent(ent.Owner, ent.Name, ent.Store)
	// if err != nil {
	// 	return nil, err
	// }

	// if ac == nil {
	// 	return nil, fmt.Errorf("no such agent: %s", name)
	// }

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config

	creator := func() (*api.Agent, error) {
		pn := api.Packname(pack + "/" + sub)

		c, err := findConfig(ac, pn)
		if err != nil {
			return nil, err
		}

		//
		c.Config = ac

		agent, err := r.NewAgent(c, pn)
		if err != nil {
			return nil, err
		}

		// embedded
		for _, v := range c.Embed {
			if a, err := r.Create(ctx, ac, api.Packname(v)); err != nil {
				return nil, err
			} else {
				agent.Embed = append(agent.Embed, a)
			}
		}

		//
		agent.Config = ac
		return agent, nil
	}

	if v, err := creator(); err == nil {
		agentCache.Add(key, v)
		return v, nil
	} else {
		return nil, err
	}
}
