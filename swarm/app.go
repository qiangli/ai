package swarm

import (
	"context"
	"fmt"
	// "maps"
	// "os"
	"path"
	"strings"
	"time"

	// "dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	// "gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"

	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/util"
)

type AgentCacheKey struct {
	// user email
	User string
	// owner meail
	Owner string
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
func (ap *AgentMaker) findAgentConfig(ac *api.AgentsConfig, pack, sub string) (*api.AgentConfig, error) {
	n := pack
	if sub != "" {
		n = pack + "/" + sub
	}
	for _, a := range ac.Agents {
		if a.Name == n {
			return a, nil
		}
	}
	return nil, fmt.Errorf("no such agent: %s", n)
}

func (ap *AgentMaker) getAgentConfig(ac *api.AgentsConfig, pack, sub string) (*api.AgentConfig, error) {
	a, err := ap.findAgentConfig(ac, pack, sub)
	if err != nil {
		return nil, err
	}

	// read the instruction
	if a.Instruction != nil {
		ps := a.Instruction.Content

		if store, ok := a.Store.(api.AssetFS); ok {
			switch {
			case strings.HasPrefix(ps, "file:"):
				parts := strings.SplitN(a.Instruction.Content, ":", 2)
				resource := strings.TrimSpace(parts[1])
				if resource == "" {
					return nil, fmt.Errorf("empty file in instruction for agent: %s", a.Name)
				}
				relPath := store.Resolve(a.BaseDir, resource)
				content, err := store.ReadFile(relPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read instruction from file %q for agent %q: %w", resource, a.Name, err)
				}
				a.Instruction.Content = string(content)
				// log.Debugf("Loaded instruction from file %q for agent %q\n", resource, a.Name)
			case strings.HasPrefix(ps, "resource:"):
				parts := strings.SplitN(a.Instruction.Content, ":", 2)
				resource := strings.TrimSpace(parts[1])
				if resource == "" {
					return nil, fmt.Errorf("empty resource name in instruction for agent %q", a.Name)
				}
				relPath := store.Resolve(a.BaseDir, resource)
				content, err := store.ReadFile(relPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read instruction from resource %q for agent %q: %w", resource, a.Name, err)
				}
				a.Instruction.Content = string(content)
			}
		}
	}
	return a, nil
}

func (ap *AgentMaker) newAgent(
	ac *api.AgentsConfig,
	c *api.AgentConfig,
	// vars *api.Vars,
	user string,
	owner string,
	// input *api.UserInput,
) (*api.Agent, error) {
	var agent = api.Agent{
		// Parent:  req.Parent,
		Owner:   owner,
		Adapter: c.Adapter,
		//
		Name:        c.Name,
		Display:     c.Display,
		Description: c.Description,
		//
		// RawInput: req.RawInput,
		//
		LogLevel: api.Quiet,
	}
	//
	agent.Arguments = c.Arguments
	// if len(req.RawInput.Arguments) > 0 {
	// 	if agent.Arguments == nil {
	// 		agent.Arguments = make(map[string]any)
	// 	}
	// 	maps.Copy(agent.Arguments, req.RawInput.Arguments)
	// }
	// agent.New = nbl(req.RawInput.New, c.New, ac.New)

	// agent.Message = nvl(req.RawInput.Message, c.Message, ac.Message)
	// agent.Format = nvl(req.RawInput.Format, c.Format, ac.Format)
	// //
	// agent.MaxTurns = nzl(req.RawInput.MaxTurns, c.MaxTurns, ac.MaxTurns, defaultMaxTurns)
	// agent.MaxTime = nzl(req.RawInput.MaxTime, c.MaxTime, ac.MaxTime, defaultMaxTime)

	// agent.MaxHistory = nzl(req.RawInput.MaxHistory, c.MaxHistory, ac.MaxHistory, defaultMaxHistory)
	// agent.MaxSpan = nzl(req.RawInput.MaxSpan, c.MaxSpan, ac.MaxSpan, defaultMaxSpan)

	//
	agent.Message = nvl(c.Message, ac.Message)
	agent.Format = nvl(c.Format, ac.Format)
	//
	agent.MaxTurns = nzl(c.MaxTurns, ac.MaxTurns, defaultMaxTurns)
	agent.MaxTime = nzl(c.MaxTime, ac.MaxTime, defaultMaxTime)

	agent.MaxHistory = nzl(c.MaxHistory, ac.MaxHistory, defaultMaxHistory)
	agent.MaxSpan = nzl(c.MaxSpan, ac.MaxSpan, defaultMaxSpan)

	// merge global vars
	agent.Environment = api.NewEnvironment()
	agent.Environment.SetEnvs(ac.Environment)
	agent.Environment.SetEnvs(c.Environment)

	// log
	agent.LogLevel = api.ToLogLevel(nvl(c.LogLevel, ac.LogLevel, "quiet"))

	// hard limit
	agent.MaxTurns = min(agent.MaxTurns, maxTurnsLimit)
	agent.MaxTime = min(agent.MaxTime, maxTimeLimit)

	// instruction
	// TODO ai trigger
	if c.Instruction != nil {
		c.Instruction.Content = strings.TrimSpace(c.Instruction.Content)
		agent.Instruction = c.Instruction
	}

	// context
	// TODO ai trigger
	context := strings.TrimSpace(nvl(c.Context, ac.Context))
	agent.Context = context

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
			// set, level := ap.resolveModelLevel(req.RawInput.Model, model)
			set, level := ap.resolveModelLevel(model)
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
					// return nil, fmt.Errorf("failed to load model: %s %s %v", req.RawInput.Model, model, err)
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
		// local
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
	if c.Flow != nil {
		var actionMap = make(map[string]*api.Action)
		for _, v := range agent.Tools {
			actionMap[v.Kit+":"+v.Name] = &api.Action{
				ID:        v.ID(),
				Name:      v.Name,
				Arguments: v.Arguments,
			}
		}
		flow := &api.Flow{
			Type:        c.Flow.Type,
			Expression:  c.Flow.Expression,
			Concurrency: c.Flow.Concurrency,
			Retry:       c.Flow.Retry,
			Script:      c.Flow.Script,
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

func (ap *AgentMaker) getAgent(owner string, pack string, asset api.AssetStore) (*api.AgentsConfig, error) {
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

	//
	ac, err := conf.LoadAgentsData([][]byte{content})
	if err != nil {
		return nil, err
	}
	if ac == nil || len(ac.Agents) == 0 {
		return nil, fmt.Errorf("invalid config. no agent defined: %s", pack)
	}

	//
	ac.Name = pack

	// agents
	for _, v := range ac.Agents {
		v.Name = ap.normalizeAgentName(pack, v.Name)
	}

	return ac, nil
}

func (ap *AgentMaker) normalizeAgentName(pack, name string) string {
	ensure := func() string {
		// pack name
		if name == "" {
			return pack
		}
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 1 {
			return pack
		}
		return pack + "/" + parts[1]
	}
	return util.NormalizedName(ensure())
}

// , auth *api.User, secrets api.SecretStore, assets api.AssetManager
func (ap *AgentMaker) CreateAgent(ctx context.Context, agent string) (*api.Agent, error) {
	// var agent = req.Name
	// var input = req.RawInput
	//

	// newAgent := func(
	// 	ac *api.AgentsConfig,
	// 	c *api.AgentConfig,
	// 	// vars *api.Vars,
	// 	user string,
	// 	owner string,
	// 	// input *api.UserInput,
	// ) (*api.Agent, error) {
	// 	var agent = api.Agent{
	// 		Parent:  req.Parent,
	// 		Owner:   owner,
	// 		Adapter: c.Adapter,
	// 		//
	// 		Name:        c.Name,
	// 		Display:     c.Display,
	// 		Description: c.Description,
	// 		//
	// 		RawInput: req.RawInput,
	// 		//
	// 		LogLevel: api.Quiet,
	// 	}
	// 	//
	// 	agent.Arguments = c.Arguments
	// 	if len(req.RawInput.Arguments) > 0 {
	// 		if agent.Arguments == nil {
	// 			agent.Arguments = make(map[string]any)
	// 		}
	// 		maps.Copy(agent.Arguments, req.RawInput.Arguments)
	// 	}
	// 	// agent.New = nbl(req.RawInput.New, c.New, ac.New)

	// 	agent.Message = nvl(req.RawInput.Message, c.Message, ac.Message)
	// 	agent.Format = nvl(req.RawInput.Format, c.Format, ac.Format)
	// 	//
	// 	agent.MaxTurns = nzl(req.RawInput.MaxTurns, c.MaxTurns, ac.MaxTurns, defaultMaxTurns)
	// 	agent.MaxTime = nzl(req.RawInput.MaxTime, c.MaxTime, ac.MaxTime, defaultMaxTime)

	// 	agent.MaxHistory = nzl(req.RawInput.MaxHistory, c.MaxHistory, ac.MaxHistory, defaultMaxHistory)
	// 	agent.MaxSpan = nzl(req.RawInput.MaxSpan, c.MaxSpan, ac.MaxSpan, defaultMaxSpan)

	// 	// merge global vars
	// 	agent.Environment = ac.Environment
	// 	if len(c.Environment) > 0 {
	// 		if agent.Environment == nil {
	// 			agent.Environment = make(map[string]any)
	// 		}
	// 		maps.Copy(agent.Environment, c.Environment)
	// 	}

	// 	// log
	// 	agent.LogLevel = api.ToLogLevel(nvl(req.RawInput.LogLevel, c.LogLevel, ac.LogLevel, "quiet"))

	// 	// hard limit
	// 	agent.MaxTurns = min(agent.MaxTurns, maxTurnsLimit)
	// 	agent.MaxTime = min(agent.MaxTime, maxTimeLimit)

	// 	// instruction
	// 	// TODO ai trigger
	// 	if c.Instruction != nil {
	// 		c.Instruction.Content = strings.TrimSpace(c.Instruction.Content)
	// 		agent.Instruction = c.Instruction
	// 	}

	// 	// context
	// 	// TODO ai trigger
	// 	context := strings.TrimSpace(nvl(c.Context, ac.Context))
	// 	agent.Context = context

	// 	// llm model set[/level]
	// 	// @model support
	// 	// flow does not require a model
	// 	model := strings.TrimSpace(nvl(c.Model, ac.Model))
	// 	if model != "" && agent.Flow == nil {
	// 		if strings.HasPrefix(model, "@") {
	// 			// defer model provider resolution
	// 			agent.Model = &api.Model{
	// 				Model: model,
	// 			}
	// 		} else {
	// 			set, level := resolveModelLevel(req.RawInput.Model, model)
	// 			// local
	// 			if set == ac.Set {
	// 				for k, v := range ac.Models {
	// 					if k == level {
	// 						agent.Model = &api.Model{
	// 							Model: v.Model,
	// 							//
	// 							Provider: nvl(v.Provider, ac.Provider),
	// 							BaseUrl:  nvl(v.BaseUrl, ac.BaseUrl),
	// 							ApiKey:   nvl(v.ApiKey, ac.ApiKey),
	// 						}
	// 					}
	// 				}
	// 			}
	// 			// load external model if not defined locally
	// 			if agent.Model == nil {
	// 				if v, err := loadModel(owner, set, level, assets); err != nil {
	// 					return nil, fmt.Errorf("failed to load model: %s %s %v", req.RawInput.Model, model, err)
	// 				} else {
	// 					agent.Model = v
	// 				}
	// 			}
	// 		}
	// 	}

	// 	// tools
	// 	funcMap := make(map[string]*api.ToolFunc)
	// 	// kit:*
	// 	for _, v := range c.Functions {
	// 		var tools []*api.ToolFunc
	// 		// local
	// 		if v, err := LoadLocalToolFunc(ac, owner, v, secrets, assets); err != nil {
	// 			return nil, err
	// 		} else {
	// 			tools = v
	// 		}
	// 		// load external kit if not defined locally
	// 		if tools == nil {
	// 			if v, err := LoadToolFunc(owner, v, secrets, assets); err != nil {
	// 				return nil, err
	// 			} else {
	// 				tools = v
	// 			}
	// 		}
	// 		for _, fn := range tools {
	// 			id := fn.ID()
	// 			if id == "" {
	// 				return nil, fmt.Errorf("agent tool ID is empty: %s", c.Name)
	// 			}
	// 			funcMap[id] = fn
	// 		}
	// 	}
	// 	var funcs []*api.ToolFunc
	// 	for _, v := range funcMap {
	// 		funcs = append(funcs, v)
	// 	}
	// 	agent.Tools = funcs

	// 	// flow
	// 	if c.Flow != nil {
	// 		var actionMap = make(map[string]*api.Action)
	// 		for _, v := range agent.Tools {
	// 			actionMap[v.Kit+":"+v.Name] = &api.Action{
	// 				ID:        v.ID(),
	// 				Name:      v.Name,
	// 				Arguments: v.Arguments,
	// 			}
	// 		}
	// 		flow := &api.Flow{
	// 			Type:        c.Flow.Type,
	// 			Expression:  c.Flow.Expression,
	// 			Concurrency: c.Flow.Concurrency,
	// 			Retry:       c.Flow.Retry,
	// 			Script:      c.Flow.Script,
	// 		}

	// 		for _, v := range c.Flow.Actions {
	// 			a, ok := actionMap[v]
	// 			if !ok {
	// 				return nil, fmt.Errorf("action missing: %s %s", agent.Name, v)
	// 			}
	// 			flow.Actions = append(flow.Actions, a)
	// 		}
	// 		agent.Flow = flow
	// 	}

	// 	return &agent, nil
	// }

	// create the agent
	// agent: owner:pack/sub
	var user = ap.sw.User.Email
	owner, pack, sub := api.AgentName(agent).Decode()
	if owner == "" {
		owner = user
	}

	// cached agent

	key := AgentCacheKey{
		User:  user,
		Owner: owner,
		Pack:  pack,
		Sub:   sub,
	}
	// return a cloned copy if found
	if v, ok := agentCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached agent: %+v", key)
		return v.Clone(), nil
	}

	var ent *api.Record
	if v, err := ap.sw.Assets.SearchAgent(owner, pack); err != nil {
		return nil, err
	} else {
		ent = v
	}
	// invalid agent
	if ent == nil && pack != "" {
		return nil, fmt.Errorf("agent not found: %s", pack)
	}

	var as api.AssetStore
	if ent == nil {
		// super agent auto selection
		pack = "agent"
		owner = user
		as = resource.NewStandardStore()
	} else {
		pack = ent.Name
		owner = ent.Owner
		as = ent.Store
	}

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config
	ac, err := ap.getAgent(owner, pack, as)
	if err != nil {
		return nil, err
	}

	if ac == nil {
		return nil, fmt.Errorf("no such agent: %s", agent)
	}

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config
	creator := func() (*api.Agent, error) {
		c, err := ap.getAgentConfig(ac, pack, sub)
		if err != nil {
			return nil, err
		}

		agent, err := ap.newAgent(ac, c, user, owner)
		if err != nil {
			return nil, err
		}

		// embedded
		for _, v := range c.Embed {
			// nomalize agent name, remove prefix "agent:""
			n := strings.TrimSpace(v)
			n = strings.ToLower(n)
			n = strings.TrimPrefix(n, "agent:")
			// nreq := &api.Request{
			// 	Name:     n,
			// 	// RawInput: req.RawInput,
			// 	RawInput: &api.UserInput{}
			// }
			if a, err := ap.CreateAgent(ctx, n); err != nil {
				return nil, err
			} else {
				agent.Embed = append(agent.Embed, a)
			}
		}
		return agent, nil
	}

	if v, err := creator(); err == nil {
		agentCache.Add(key, v)
		return v, nil
	} else {
		return nil, err
	}
}

func (ap *AgentMaker) resolveModelLevel(model string) (string, string) {
	// // models takes precedence over model
	// split := func() (string, string) {
	// 	// models: alias[/level]
	// 	// alias, level := split2(models, "/", "")

	// 	// // model: [alias/]level
	// 	// parts := strings.SplitN(model, "/", 2)
	// 	// if len(parts) == 2 {
	// 	// 	return nvl(alias, parts[0]), nvl(level, parts[1])
	// 	// }

	// 	// return nvl(alias, "default"), nvl(level, model)
	// }

	// alias/level
	// alias, level := split()
	alias, level := split2(model, "/", "any")
	return alias, level
}
