package swarm

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
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
const defaultMaxHistory = 3

type ConfigLoader struct {
	data []byte
	rte  *api.ActionRTEnv
}

func NewConfigLoader(rte *api.ActionRTEnv) *ConfigLoader {
	return &ConfigLoader{
		rte: rte,
	}
}

func (r *ConfigLoader) LoadContent(s string) error {
	v, err := api.LoadURIContent(r.rte.Workspace, s)
	if err != nil {
		return err
	}
	r.data = []byte(v)
	return nil
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

func (r *ConfigLoader) LoadAgentConfig(packname api.Packname) (*api.AppConfig, error) {
	pack, sub := packname.Decode()
	if pack == "" {
		return nil, fmt.Errorf("pack is required")
	}

	if r.data != nil {
		ac, err := conf.LoadAgentsData([][]byte{r.data})
		if err != nil {
			return nil, err
		}

		for _, v := range ac.Agents {
			if v.Name == sub {
				ac.Pack = pack
				return ac, nil
			}
		}
		// continue to find
	}

	ac, err := r.rte.Assets.FindAgent(r.rte.User.Email, pack)
	if err != nil {
		return nil, err
	}
	if ac == nil {
		return nil, fmt.Errorf("could not find the config for pack: %s", pack)
	}

	for _, v := range ac.Agents {
		if v.Name == sub {
			ac.Pack = pack
			return ac, nil
		}
	}
	return nil, fmt.Errorf("config not found for: %s", packname)
}

func (r *ConfigLoader) LoadToolConfig(kn api.Kitname) (*api.AppConfig, error) {
	kit, name := kn.Decode()

	if kit == "" {
		return nil, fmt.Errorf("kit is required")
	}
	if name == "" {
		name = kit
	}

	if r.data != nil {
		tc, err := conf.LoadToolData([][]byte{r.data})
		if err != nil {
			return nil, err
		}

		for _, v := range tc.Tools {
			if v.Name == name {
				tc.Kit = kit
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
		if v.Name == name {
			tc.Kit = kit
			return tc, nil
		}
	}

	return nil, fmt.Errorf("could not find config for: %s", name)
}

func (r *ConfigLoader) CreateTool(tid string) (*api.ToolFunc, error) {
	kn := api.Kitname(tid)
	kit, name := kn.Decode()
	if kit == "" {
		return nil, fmt.Errorf("invalid tool name: %v. missing kit", tid)
	}

	// /agent:
	if kit == string(api.ToolTypeAgent) {
		pn := api.Packname(name).Clean()
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

		if name == "" {
			name = kit
		}
		for _, v := range tools {
			if v.Name == name {
				return v, nil
			}
		}
	}
	return nil, nil
}

func (r *ConfigLoader) NewAgent(c *api.AgentConfig, pn api.Packname) (*api.Agent, error) {

	ac := c.Config

	// new agent
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
		//
		Parameters: c.Parameters,
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

	//
	// ac.Arguments
	maps.Copy(args, c.Arguments)
	agent.Arguments.SetArgs(args)

	// merge global vars
	agent.Environment = api.NewEnvironment()
	// agent.Environment.AddEnvs(ac.Environment)
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
			set, level := api.Setlevel(model).Decode()
			// local
			if set == ac.Set {
				for k, v := range ac.Models {
					if k == level {
						agent.Model = &api.Model{
							Set:   set,
							Level: level,
							//
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

	return &agent, nil
}

// create agent (class) from config
func (r *ConfigLoader) Create(ctx context.Context, packname api.Packname) (*api.Agent, error) {
	findConfig := func(ac *api.AppConfig, pack, sub string) (*api.AgentConfig, error) {
		for _, a := range ac.Agents {
			if ac.Pack == pack && sub == a.Name {
				return a, nil
			}
		}

		return nil, fmt.Errorf("agent not found: %s/%s", pack, sub)
	}

	// create the agent
	// agent: pack/sub
	// var user = ap.sw.User.Email
	pack, sub := packname.Clean().Decode()

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

	ac, err := r.LoadAgentConfig(packname)
	if err != nil {
		return nil, err
	}

	creator := func() (*api.Agent, error) {
		c, err := findConfig(ac, pack, sub)
		if err != nil {
			return nil, err
		}

		//
		c.Config = ac
		pn := api.Packname(pack + "/" + sub)
		agent, err := r.NewAgent(c, pn)
		if err != nil {
			return nil, err
		}

		// embedded
		for _, v := range c.Embed {
			if a, err := r.Create(ctx, api.Packname(v)); err != nil {
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
