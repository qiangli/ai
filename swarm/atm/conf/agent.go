package conf

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/resource"
	"github.com/qiangli/ai/swarm/util"
)

type AgentsConfigCacheKey struct {
	// owner meail
	Owner string
	// agent name
	Name string
}

var (
	agentCache = expirable.NewLRU[AgentsConfigCacheKey, *api.AgentsConfig](10000, nil, time.Second*180)
)

// max hard upper limits
const maxTurnsLimit = 50
const maxTimeLimit = 300 // 5 min

const defaultMaxTurns = 8
const defaultMaxTime = 180 // 3 min

func ListAgents(owner string, am api.AssetManager) (map[string]*api.AgentsConfig, error) {
	return am.ListAgent(owner)
}

// ensure name is correct
// <agent>/sub
// duplicate name is not checked
func normalizeAgentName(name, sub string) string {
	ensure := func() string {
		// pack name
		if sub == "" {
			return name
		}
		parts := strings.SplitN(sub, "/", 2)
		if len(parts) == 1 {
			return name
		}
		return name + "/" + parts[1]
	}
	return util.NormalizedName(ensure())
}

func loadAgentsATM(owner string, as api.ATMSupport, packs map[string]*api.AgentsConfig) error {
	recs, err := as.ListAgents(owner)
	if err != nil {
		return err
	}

	// not found
	if len(recs) == 0 {
		return nil
	}

	for _, v := range recs {
		ac, err := LoadAgentsData([][]byte{[]byte(v.Content)})
		if err != nil {
			return err
		}
		if ac == nil || len(ac.Agents) == 0 {
			return fmt.Errorf("invalid config. no agent defined: %s", v.Name)
		}

		ac.Name = strings.ToLower(v.Name)
		if _, ok := packs[ac.Name]; ok {
			continue
		}

		// agents
		for _, v := range ac.Agents {
			v.Name = normalizeAgentName(ac.Name, v.Name)
			//
			v.Store = as
		}
		packs[ac.Name] = ac
	}

	return nil
}

func loadAgentsAsset(as api.AssetFS, root string, packs map[string]*api.AgentsConfig) error {
	dirs, err := as.ReadDir(root)
	if err != nil {
		return err
	}

	// not found
	if len(dirs) == 0 {
		return nil
	}

	for _, v := range dirs {
		if !v.IsDir() {
			continue
		}
		base := path.Join(root, v.Name())
		name := path.Join(base, "agent.yaml")
		content, err := as.ReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read agent asset %s: %w", v.Name(), err)
		}
		if len(content) == 0 {
			continue
		}

		ac, err := LoadAgentsData([][]byte{content})
		if err != nil {
			return err
		}
		if ac == nil || len(ac.Agents) == 0 {
			return fmt.Errorf("invalid config. no agent defined: %s", name)
		}

		ac.Name = strings.ToLower(v.Name())

		if _, ok := packs[ac.Name]; ok {
			continue
		}

		// keep store loader for loading extra resources later
		for _, v := range ac.Agents {
			v.Name = normalizeAgentName(name, v.Name)

			v.Store = as
			v.BaseDir = base
		}
		packs[ac.Name] = ac
	}

	return nil
}

// LoadAgentsData loads the agent configuration from the provided YAML data.
func LoadAgentsData(data [][]byte) (*api.AgentsConfig, error) {
	merged := &api.AgentsConfig{}

	for _, v := range data {
		cfg := &api.AgentsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	return merged, nil
}

func getAgent(owner string, pack string, asset api.AssetStore) (*api.AgentsConfig, error) {
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
	ac, err := LoadAgentsData([][]byte{content})
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
		v.Name = normalizeAgentName(pack, v.Name)
	}

	if ac.MaxTurns == 0 {
		ac.MaxTurns = defaultMaxTurns
	}
	if ac.MaxTime == 0 {
		ac.MaxTime = defaultMaxTime
	}
	// upper limit
	ac.MaxTurns = min(ac.MaxTurns, maxTurnsLimit)
	ac.MaxTime = min(ac.MaxTime, maxTimeLimit)

	return ac, nil
}

func CreateAgent(vars *api.Vars, auth *api.User, secrets api.SecretStore, assets api.AssetManager, req *api.Request) (*api.Agent, error) {
	//
	findAgentConfig := func(ac *api.AgentsConfig, pack, sub string) (*api.AgentConfig, error) {
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

	getAgentConfig := func(ac *api.AgentsConfig, pack, sub string) (*api.AgentConfig, error) {
		a, err := findAgentConfig(ac, pack, sub)
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

	newAgent := func(
		ac *api.AgentsConfig,
		c *api.AgentConfig,
		vars *api.Vars,
		user string,
		owner string,
		input *api.UserInput,
	) (*api.Agent, error) {
		agent := api.Agent{
			Owner:   owner,
			Adapter: c.Adapter,
			//
			Name:        c.Name,
			Display:     c.Display,
			Description: c.Description,
			//
			Instruction: c.Instruction,
			//
			RawInput: input,
			//
			MaxTurns: nzl(vars.Config.MaxTurns, c.MaxTurns, ac.MaxTurns),
			MaxTime:  nzl(vars.Config.MaxTime, c.MaxTime, ac.MaxTime),
			//
			Message:  nvl(vars.Config.Message, c.Message, ac.Message),
			Format:   nvl(vars.Config.Format, c.Format, ac.Format),
			New:      nbl(vars.Config.New, c.New, ac.New),
			LogLevel: api.Quiet,
			//
			Dependencies: c.Dependencies,
			//
			Config: ac,
		}

		// log
		agent.LogLevel = api.ToLogLevel(nvl(vars.Config.LogLevel, c.LogLevel, ac.LogLevel, "quiet"))

		// llm model
		model := nvl(c.Model, ac.Model)
		if v, err := loadModel(auth, owner, vars.Config.Models, model, secrets, assets); err != nil {
			return nil, fmt.Errorf("failed to load model: %s %s %v", vars.Config.Models, model, err)
		} else {
			agent.Model = v
		}

		// tools
		funcMap := make(map[string]*api.ToolFunc)
		for _, v := range c.Functions {
			// kit:*
			tools, err := loadToolFunc(owner, v, secrets, assets)
			if err != nil {
				return nil, err
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

		if c.Advices != nil {
			// TODO
			return nil, fmt.Errorf("advice no supported: %+v", c.Advices)
		}
		if c.Entrypoint != "" {
			// TODO
			return nil, fmt.Errorf("entrypoint not supported: %s", c.Entrypoint)
		}

		return &agent, nil
	}

	// create the agent
	// read config and create agent
	var user = auth.Email
	// @<[owner:]agent>
	owner, agent := splitOwnerAgent(req.Agent)
	// agent: [pack/]sub
	pack, sub := split2(agent, "/", "")

	// name: [owner:]agent
	var ent *api.Record
	if v, err := assets.SearchAgent(owner, pack); err != nil {
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
	ac, err := getAgent(owner, pack, as)
	if err != nil {
		return nil, err
	}

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config
	creator := func() (*api.Agent, error) {
		c, err := getAgentConfig(ac, pack, sub)
		if err != nil {
			return nil, err
		}

		agent, err := newAgent(ac, c, vars, user, owner, req.RawInput)
		if err != nil {
			return nil, err
		}

		return agent, nil
	}

	return creator()
}
