package swarm

import (
	"context"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"strings"

	// "github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
)

type AgentCacheKey struct {
	// user email
	User string
	// agent
	Pack string
	Sub  string
}

// TODO can not be cached - cache config only?
// var (
// 	agentCache = expirable.NewLRU[AgentCacheKey, *api.Agent](10000, nil, time.Second*900)
// )

const maxTurnsLimit = 100
const maxTimeLimit = 900 // 15 min

type ConfigLoader struct {
	base   string
	assets api.AssetStore
	data   []byte

	vars *api.Vars
}

func NewConfigLoader(vars *api.Vars) *ConfigLoader {
	return &ConfigLoader{
		vars: vars,
	}
}

func (r *ConfigLoader) LoadContent(src string) error {
	var ws = r.vars.Workspace
	var content []byte
	if strings.HasPrefix(src, "data:") {
		v, err := api.DecodeDataURL(src)
		if err != nil {
			return err
		}
		content = []byte(v)
	} else {
		var f = src
		if strings.HasPrefix(f, "file:") {
			v, err := url.Parse(f)
			if err != nil {
				return err
			}
			f = v.Path
		}
		data, err := ws.ReadFile(f, nil)
		if err != nil {
			return err
		}
		content = data
		r.base = filepath.Dir(f)
		r.assets = ws
	}
	r.data = content
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

// Load assets for context, instrution, message, and script
func (r *ConfigLoader) ResolveAssets(ac *api.AppConfig) (*api.AppConfig, error) {
	for _, a := range ac.Agents {
		if strings.HasPrefix(a.Context, "asset:") {
			v, err := atm.LoadAsset(ac.Store, ac.BaseDir, a.Context[6:])
			if err != nil {
				return nil, err
			}
			a.Context = v
		}
		if strings.HasPrefix(a.Instruction, "asset:") {
			v, err := atm.LoadAsset(ac.Store, ac.BaseDir, a.Instruction[6:])
			if err != nil {
				return nil, err
			}
			a.Instruction = v
		}
		if strings.HasPrefix(a.Message, "asset:") {
			v, err := atm.LoadAsset(ac.Store, ac.BaseDir, a.Message[6:])
			if err != nil {
				return nil, err
			}
			a.Message = v
		}
	}
	for _, t := range ac.Tools {
		if t.Body != nil {
			s := t.Body.Script
			if strings.HasPrefix(s, "asset:") {
				v, err := atm.LoadAsset(ac.Store, ac.BaseDir, s[6:])
				if err != nil {
					return nil, err
				}
				t.Body.Script = v
			}
		}
	}
	return ac, nil
}

func (r *ConfigLoader) LoadAgentConfig(packname api.Packname) (*api.AppConfig, error) {
	pack, sub := packname.Decode()
	if pack == "" {
		return nil, fmt.Errorf("pack is required")
	}

	// from script arg
	if r.data != nil {
		ac, err := conf.LoadAgentsData([][]byte{r.data})
		if err != nil {
			return nil, err
		}
		if ac.Pack == pack {
			for _, v := range ac.Agents {
				if v.Name == sub {
					ac.BaseDir = r.base
					ac.Store = r.assets
					return r.ResolveAssets(ac)
				}
			}
		}
		// continue to find
	}

	ac, err := r.vars.Assets.FindAgent(r.vars.User.Email, pack)
	if err != nil {
		return nil, err
	}
	if ac == nil {
		return nil, fmt.Errorf("could not find the config for pack: %s", pack)
	}

	for _, v := range ac.Agents {
		if v.Name == sub {
			ac.Pack = pack
			return r.ResolveAssets(ac)
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

	// from script arg
	if r.data != nil {
		tc, err := conf.LoadToolData([][]byte{r.data})
		if err != nil {
			return nil, err
		}

		for _, v := range tc.Tools {
			if v.Name == name {
				tc.Kit = kit
				tc.BaseDir = r.base
				tc.Store = r.assets
				return r.ResolveAssets(tc)
			}
		}
		// continue to find
	}

	tc, err := r.vars.Assets.FindToolkit(r.vars.User.Email, kit)
	if err != nil {
		return nil, err
	}
	if tc == nil {
		return nil, fmt.Errorf("could not find the config for kit: %s", kit)
	}

	for _, v := range tc.Tools {
		if v.Name == name {
			tc.Kit = kit
			return r.ResolveAssets(tc)
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
		pack, sub := pn.Decode()
		v, err := conf.LoadAgentTool(ac, pack, sub)
		if err != nil {
			return nil, err
		}
		if len(v) != 1 {
			return nil, fmt.Errorf("Error loading agent tool: %s/%s", pack, sub)
		}
		return v[0], err
	} else {
		// /kit:tool
		tc, err := r.LoadToolConfig(kn)
		if err != nil {
			return nil, err
		}

		tools, err := conf.LoadTools(tc, r.vars.User.Email, r.vars.Secrets)
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
		//
		Config: ac,
	}
	//
	args := make(map[string]any)
	maps.Copy(args, ac.ToMap())
	maps.Copy(args, c.ToMap())

	maxTurns := nzl(c.MaxTurns, ac.MaxTurns)
	maxTime := nzl(c.MaxTime, ac.MaxTime)

	// hard limit
	maxTurns = min(maxTurns, maxTurnsLimit)
	maxTime = min(maxTime, maxTimeLimit)

	if maxTurns > 0 {
		args["max_turns"] = maxTurns
	}
	if maxTime > 0 {
		args["max_time"] = maxTime
	}

	maxHistory := nzl(c.MaxHistory, ac.MaxHistory)
	maxSpan := nzl(c.MaxSpan, ac.MaxSpan)

	if maxHistory > 0 {
		args["max_history"] = maxHistory
	}
	if maxSpan > 0 {
		args["max_span"] = maxSpan
	}

	// log
	logLevel := nvl(c.LogLevel, ac.LogLevel)
	if logLevel != "" {
		args["log_level"] = logLevel
	}

	//
	// ac.Arguments
	maps.Copy(args, c.Arguments)
	agent.Arguments.SetArgs(args)

	// merge global vars
	agent.Environment = api.NewEnvironment()
	// agent.Environment.AddEnvs(ac.Environment)
	agent.Environment.AddEnvs(c.Environment)

	// llm model set[/level]
	model := strings.TrimSpace(nvl(c.Model, ac.Model))

	if model != "" {
		// if strings.HasPrefix(model, "@") {
		// 	// defer model provider resolution
		// 	agent.Model = &api.Model{
		// 		Model: model,
		// 	}
		// } else {
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
					break
				}
			}
		}
		// load external model if not defined locally
		if agent.Model == nil {
			if v, err := conf.LoadModel(r.vars.User.Email, set, level, r.vars.Assets); err != nil {
				return nil, fmt.Errorf("failed to load model: %s %v", model, err)

			} else {
				agent.Model = v
			}
		}
		// }
	}

	// tools
	// dedup
	funcMap := make(map[string]*api.ToolFunc)
	// kit:*
	var tools []*api.ToolFunc
	for _, v := range c.Functions {
		// all (standard) - not including local custom tools
		if v == "kit:*" {
			all, err := r.loadAllTools()
			if err != nil {
				return nil, err
			}
			tools = append(tools, all...)
			continue
		}
		// all default agent as tool - not including agent tools that are defined under tools.
		if v == "agent:*" {
			all, err := r.loadAllAgentTools()
			if err != nil {
				return nil, err
			}
			tools = append(tools, all...)
			continue
		}
		// local scope
		if v, err := conf.LoadLocalToolFunc(ac, r.vars.User.Email, v, r.vars.Secrets, r.vars.Assets); err == nil {
			tools = append(tools, v...)
			continue
		}
		// load external kit if not defined locally
		if v, err := conf.LoadToolFunc(r.vars.User.Email, v, r.vars.Secrets, r.vars.Assets); err == nil {
			tools = append(tools, v...)
		}
	}

	// dedup
	for _, fn := range tools {
		id := fn.ID()
		if id == "" {
			return nil, fmt.Errorf("agent tool ID is empty: %s", c.Name)
		}
		funcMap[id] = fn
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

	// // cached agent
	// key := AgentCacheKey{
	// 	User: r.vars.User.Email,
	// 	Pack: pack,
	// 	Sub:  sub,
	// }

	// // return a cloned copy if found
	// if v, ok := agentCache.Get(key); ok {
	// 	return v.Clone(), nil
	// }

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
		// agentCache.Add(key, v)
		return v, nil
	} else {
		return nil, err
	}
}

// load all kit:* tools defined under "tools" in a yaml including agent tools.
func (r *ConfigLoader) loadAllTools() ([]*api.ToolFunc, error) {
	owner := r.vars.User.Email
	tools, err := r.vars.Assets.ListToolkit(owner)
	if err != nil {
		return nil, err
	}
	var funcs []*api.ToolFunc
	for _, tc := range tools {
		v, err := conf.LoadTools(tc, owner, r.vars.Secrets)
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, v...)
	}
	return funcs, nil
}

// load all agent:* as tools defined under "agents" in a yaml
func (r *ConfigLoader) loadAllAgentTools() ([]*api.ToolFunc, error) {
	agents, err := r.vars.Assets.ListAgent(r.vars.User.Email)
	if err != nil {
		return nil, err
	}

	var funcs []*api.ToolFunc
	for _, ac := range agents {
		for _, sub := range ac.Agents {
			v, err := conf.LoadAgentTool(ac, ac.Pack, sub.Name)
			if err != nil {
				// ignore error?
				continue
			}
			funcs = append(funcs, v...)
		}
	}
	return funcs, nil
}
