package swarm

import (
	"context"
	"fmt"
	"maps"
	"path"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"

	"github.com/qiangli/ai/swarm/log"
)

type AgentCacheKey struct {
	// user email
	User string
	// agent
	Pack string
	Sub  string
}

var (
	agentCache = expirable.NewLRU[AgentCacheKey, *api.Agent](10000, nil, time.Second*900)
)

const maxTurnsLimit = 100
const maxTimeLimit = 900 // 15 min

const defaultMaxTurns = 50
const defaultMaxTime = 600 // 10 min

const defaultMaxSpan = 1440 // 24 hours
const defaultMaxHistory = 1

type AgentMaker struct {
	sw *Swarm
}

func NewAgentMaker(sw *Swarm) *AgentMaker {
	return &AgentMaker{
		sw: sw,
	}
}
func findAgentConfig(ac *api.AppConfig, pack, sub string) (*api.AgentConfig, error) {
	// pn := pack
	// if sub != "" {
	// 	pn = pack + "/" + sub
	// }
	pn := api.Packname(pack + "/" + sub)
	for _, a := range ac.Agents {
		if pn.Equal(a.Name) {
			return a, nil
		}
		// if a.Name == pn {
		// 	return a, nil
		// }
		// primary entry
		// if a.Name == pack && sub == "" {
		// 	return a, nil
		// }
	}
	return nil, fmt.Errorf("no such agent: %s", pn)
}

func getAgentConfig(ac *api.AppConfig, pack, sub string) (*api.AgentConfig, error) {
	a, err := findAgentConfig(ac, pack, sub)
	if err != nil {
		return nil, err
	}

	// // read the instruction
	// if a.Instruction != "" {
	// 	ps := a.Instruction

	// 	if store, ok := a.Store.(api.AssetFS); ok {
	// 		switch {
	// 		case strings.HasPrefix(ps, "file:"):
	// 			parts := strings.SplitN(a.Instruction, ":", 2)
	// 			resource := strings.TrimSpace(parts[1])
	// 			if resource == "" {
	// 				return nil, fmt.Errorf("empty file in instruction for agent: %s", a.Name)
	// 			}
	// 			relPath := store.Resolve(a.BaseDir, resource)
	// 			content, err := store.ReadFile(relPath)
	// 			if err != nil {
	// 				return nil, fmt.Errorf("failed to read instruction from file %q for agent %q: %w", resource, a.Name, err)
	// 			}
	// 			a.Instruction = string(content)
	// 			// log.Debugf("Loaded instruction from file %q for agent %q\n", resource, a.Name)
	// 		case strings.HasPrefix(ps, "resource:"):
	// 			parts := strings.SplitN(a.Instruction, ":", 2)
	// 			resource := strings.TrimSpace(parts[1])
	// 			if resource == "" {
	// 				return nil, fmt.Errorf("empty resource name in instruction for agent %q", a.Name)
	// 			}
	// 			relPath := store.Resolve(a.BaseDir, resource)
	// 			content, err := store.ReadFile(relPath)
	// 			if err != nil {
	// 				return nil, fmt.Errorf("failed to read instruction from resource %q for agent %q: %w", resource, a.Name, err)
	// 			}
	// 			a.Instruction = string(content)
	// 		}
	// 	}
	// }
	return a, nil
}

func (ap *AgentMaker) newAgent(
	ac *api.AppConfig,
	c *api.AgentConfig,
	owner string,
) (*api.Agent, error) {
	var agent = api.Agent{
		Pack:    ac.Pack,
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
	// if model != "" && agent.Flow == nil {
	if model != "" {
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
				if v, err := conf.LoadModel(owner, set, level, ap.sw.Assets); err != nil {
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
		if v, err := conf.LoadLocalToolFunc(ac, owner, v, ap.sw.Secrets, ap.sw.Assets); err != nil {
			return nil, err
		} else {
			tools = v
		}
		// load external kit if not defined locally
		if tools == nil {
			if v, err := conf.LoadToolFunc(owner, v, ap.sw.Secrets, ap.sw.Assets); err != nil {
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
	// if c.Flow != nil {
	// 	var actionMap = make(map[string]*api.Action)
	// 	for _, v := range agent.Tools {
	// 		actionMap[v.Kit+":"+v.Name] = api.NewAction(
	// 			v.ID(),
	// 			v.Name,
	// 			v.Arguments,
	// 		)
	// 	}
	// 	flow := &api.Flow{
	// 		Type:   c.Flow.Type,
	// 		Script: c.Flow.Script,
	// 	}

	// 	for _, v := range c.Flow.Actions {
	// 		a, ok := actionMap[v]
	// 		if !ok {
	// 			return nil, fmt.Errorf("action missing: %s %s", agent.Name, v)
	// 		}
	// 		flow.Actions = append(flow.Actions, a)
	// 	}
	// 	agent.Flow = flow
	// }

	return &agent, nil
}

func (ap *AgentMaker) getAgent(owner string, pack string, asset api.AssetStore) (*api.AppConfig, error) {
	var content []byte
	if as, ok := asset.(api.ATMSupport); ok {
		if v, err := as.RetrieveAgent(owner, pack); err != nil {
			return nil, err
		} else {
			content = []byte(v.Content)
		}
	} else if as, ok := asset.(api.AssetFS); ok {
		if v, err := as.ReadFile(path.Join("agents", pack, "agent.yaml")); err != nil {
			return nil, err
		} else {
			content = v

		}
	}

	if len(content) == 0 {
		return nil, nil
	}

	return ap.loadAgent(pack, content)
}

func (ap *AgentMaker) loadAgent(pack string, content []byte) (*api.AppConfig, error) {
	ac, err := conf.LoadAgentsData([][]byte{content})
	if err != nil {
		return nil, err
	}
	if ac == nil || len(ac.Agents) == 0 {
		return nil, fmt.Errorf("invalid config. no agent defined: %s", pack)
	}

	// correct pack name
	ac.Name = pack
	ac.RawContent = content

	// normalize agent name
	for _, v := range ac.Agents {
		v.Name = conf.NormalizePackname(pack, v.Name)
	}

	return ac, nil
}

// create agent (class) from config
func (ap *AgentMaker) Create(ctx context.Context, name string) (*api.Agent, error) {
	// create the agent
	// agent: pack/sub
	// var user = ap.sw.User.Email
	pack, sub := api.Packname(name).Decode()

	//
	if pack == "" {
		return nil, fmt.Errorf("missing agent pack")
	}

	// cached agent
	key := AgentCacheKey{
		User: ap.sw.User.Email,
		Pack: pack,
		Sub:  sub,
	}
	// return a cloned copy if found
	if v, ok := agentCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached agent: %+v", key)
		return v.Clone(), nil
	}

	var ent *api.Record
	if v, err := ap.sw.Assets.SearchAgent(ap.sw.User.Email, pack); err != nil {
		return nil, err
	} else {
		ent = v
	}
	// invalid agent
	if ent == nil && pack != "" {
		return nil, fmt.Errorf("agent not found: %s", pack)
	}

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config
	ac, err := ap.getAgent(ent.Owner, ent.Name, ent.Store)
	if err != nil {
		return nil, err
	}

	if ac == nil {
		return nil, fmt.Errorf("no such agent: %s", name)
	}

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config
	creator := func() (*api.Agent, error) {
		c, err := getAgentConfig(ac, pack, sub)
		if err != nil {
			return nil, err
		}

		agent, err := ap.newAgent(ac, c, ent.Owner)
		if err != nil {
			return nil, err
		}

		// embedded
		for _, v := range c.Embed {
			if a, err := ap.Create(ctx, v); err != nil {
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

func (ap *AgentMaker) CreateFrom(ctx context.Context, name string, content []byte) (*api.Agent, error) {
	pack, sub := api.Packname(name).Decode()

	c, err := ap.Creator(ap.sw.agentMaker.Create, ap.sw.User.Email, pack, content)
	if err != nil {
		return nil, err
	}

	var pn = pack
	if sub != "" {
		pn = pack + "/" + sub
	}

	return c(ctx, pn)
}

func (ap *AgentMaker) Creator(parent api.Creator, owner string, pack string, data []byte) (api.Creator, error) {
	ac, err := ap.loadAgent(pack, data)
	if err != nil {
		return nil, err
	}

	var creator api.Creator
	creator = func(ctx context.Context, name string) (*api.Agent, error) {
		pack, sub := api.Packname(name).Decode()
		c, err := getAgentConfig(ac, pack, sub)
		if err != nil {
			if parent == nil {
				return nil, err
			}
			return parent(ctx, name)
		}

		agent, err := ap.newAgent(ac, c, owner)
		if err != nil {
			return nil, err
		}

		// embedded
		for _, v := range c.Embed {
			if a, err := creator(ctx, v); err != nil {
				return nil, err
			} else {
				agent.Embed = append(agent.Embed, a)
			}
		}

		// save for ai:get_agent_config
		agent.Config = ac
		return agent, nil
	}

	return creator, nil
}

func resolveModelLevel(model string) (string, string) {
	alias, level := split2(model, "/", "any")
	return alias, level
}
