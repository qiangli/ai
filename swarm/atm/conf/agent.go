package conf

import (
	// "context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
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

func ListAgents(user string) (map[string]*api.AgentsConfig, error) {
	own, err := listActiveAgentsConfig(user)
	if err != nil {
		return nil, err
	}

	var agents = make(map[string]*api.AgentsConfig)
	for k, v := range standardAgents {
		agents[k] = v
	}
	for k, v := range own {
		agents[k] = v
	}

	return agents, nil
}

func loadAgentsConfig(user *api.User, owner, name string) (*api.AgentsConfig, error) {
	if name == "" {
		name = "agent"
	}

	key := AgentsConfigCacheKey{
		Owner: owner,
		Name:  name,
	}
	if v, ok := agentCache.Get(key); ok {
		return v, nil
	}

	ac, err := retrieveActiveAgentsConfig(owner, name)
	if err != nil {
		return nil, err
	}
	// found
	if ac != nil {
		agentCache.Add(key, ac)
		return ac, nil
	}

	// default
	// not reachable if shared member login
	// agent name should have been auto set
	// if !user.IsMemberLogin() {
	if ac, ok := standardAgents[name]; ok {
		return ac, nil
	}
	// }
	return nil, fmt.Errorf("agent not found: %s", name)
}

// user: user email address
// owner: partial email
// name: agent name
func searchAgent(
	user string,
	owner string,
	name string,
) (*api.Agent, error) {
	if owner == "" && name == "" {
		return nil, nil
	}

	//
	// search order: own, standard, shared
	// search own first so user could override standard agents
	// shared agents are searched to avoid supprises since potential multiple duplicates
	// @owner:agent could be used to request a specific agent if desired.
	if owner == "" || owner == user {
		// if v, found, err := db.GetActiveAgentByName(user, name); err != nil {
		// 	return nil, err
		// } else if found {
		// 	return v, nil
		// }

		// // TODO standard agent table?
		// if slices.Contains(standardAgentNames, name) {
		// 	return &entity.Agent{
		// 		Owner: user,
		// 		Name:  name,
		// 	}, nil
		// }
	}

	// owner could be empty or partial match
	// if v, found, err := db.SearchActiveSharedAgentByOwner(user, owner, name); err != nil {
	// 	return nil, err
	// } else if found {
	// 	return v, nil
	// }

	// own/shared agents not found
	return nil, nil
}

// ensure name is correct
// <agent>/sub
// duplicate name is not checked
func normalizeAgentName(name, sub string) string {
	ensure := func() string {
		// group name
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

// owner: owner email address
// name: agent name
// return nil if not found
func retrieveActiveAgentsConfig(
	owner string,
	name string,
) (*api.AgentsConfig, error) {
	// var agent *entity.Agent
	// if v, found, err := db.GetActiveAgentByName(owner, name); err != nil {
	// 	return nil, err
	// } else if found {
	// 	agent = v
	// }
	// // not found
	// if agent == nil {
	// 	return nil, nil
	// }
	// []byte(agent.Content)
	var content []byte
	ac, err := LoadAgentsData([][]byte{content})
	if err != nil {
		return nil, err
	}
	if ac == nil || len(ac.Agents) == 0 {
		return nil, fmt.Errorf("invalid config. no agent defined: %s", name)
	}

	//
	ac.Name = name

	// agents
	for _, v := range ac.Agents {
		v.Name = normalizeAgentName(name, v.Name)
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

func listActiveAgentsConfig(
	owner string,
) (map[string]*api.AgentsConfig, error) {
	// var agents []*entity.Agent
	// if v, err := db.ListActiveAgents(owner); err != nil {
	// 	return nil, err
	// } else {
	// 	agents = v
	// }
	// // not found
	// if len(agents) == 0 {
	// 	return nil, nil
	// }

	var acs = make(map[string]*api.AgentsConfig)
	// for _, agent := range agents {
	// 	name := agent.Name
	// 	ac, err := LoadAgentsData([][]byte{[]byte(agent.Content)})
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	if ac == nil || len(ac.Agents) == 0 {
	// 		return nil, fmt.Errorf("invalid config. no agent defined: %s", name)
	// 	}

	// 	// agents
	// 	for _, v := range ac.Agents {
	// 		v.Name = normalizeAgentName(name, v.Name)
	// 	}
	// 	acs[name] = ac
	// }

	return acs, nil
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

func LoadAgentsAsset(as api.AssetStore, root string, groups map[string]*api.AgentsConfig) error {
	dirs, err := as.ReadDir(root)
	if err != nil {
		return fmt.Errorf("failed to read agent resource directory: %v", err)
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		base := path.Join(root, dir.Name())
		name := path.Join(base, "agent.yaml")
		f, err := as.ReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read agent asset %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			// log.Debugf("agent file is empty %s\n", name)
			continue
		}
		group, err := LoadAgentsData([][]byte{f})
		if err != nil {
			return fmt.Errorf("failed to load agent data from %s: %w", dir.Name(), err)
		}
		if group == nil {
			// log.Debugf("no agent found in %s\n", dir.Name())
			continue
		}
		// group.BaseDir = base
		// use the name of the directory as the group name if not specified
		if group.Name == "" {
			group.Name = dir.Name()
		}
		if _, exists := groups[group.Name]; exists {
			// log.Debugf("duplicate agent name found: %s in %s, skipping\n", group.Name, dir.Name())
			continue
		}

		// keep store loader for loading extra resources later
		for _, v := range group.Agents {
			v.Store = as
			v.BaseDir = base
		}

		groups[group.Name] = group
	}

	return nil
}

func CreateAgent(vars *api.Vars, auth *api.User, secrets api.SecretStore, req *api.Request) (*api.Agent, error) {
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

			switch {
			case strings.HasPrefix(ps, "file:"):
				parts := strings.SplitN(a.Instruction.Content, ":", 2)
				resource := strings.TrimSpace(parts[1])
				if resource == "" {
					return nil, fmt.Errorf("empty file in instruction for agent: %s", a.Name)
				}
				relPath := a.Store.Resolve(a.BaseDir, resource)
				content, err := a.Store.ReadFile(relPath)
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
				relPath := a.Store.Resolve(a.BaseDir, resource)
				content, err := a.Store.ReadFile(relPath)
				if err != nil {
					return nil, fmt.Errorf("failed to read instruction from resource %q for agent %q: %w", resource, a.Name, err)
				}
				a.Instruction.Content = string(content)
				// log.Debugf("Loaded instruction from resource %q for agent %q\n", resource, a.Name)
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
		if v, err := loadModel(auth, owner, vars.Config.Models, model, secrets); err != nil {
			return nil, fmt.Errorf("failed to load model: %s %s %v", vars.Config.Models, model, err)
		} else {
			agent.Model = v
		}

		// tools
		funcMap := make(map[string]*api.ToolFunc)
		for _, v := range c.Functions {
			// kit:*
			tools, err := loadToolFunc(owner, v, secrets)
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

	// read config and create agent
	var user = auth.Email

	// @<[partial owner:]agent>
	owner, agentName := splitOwnerAgent(req.Agent)

	// agent: [pack/]sub
	pack, sub := split2(agentName, "/", "")

	// name: [partial owner:]agent
	// var ent *entity.Agent
	// if v, err := searchAgent(user, owner, pack); err != nil {
	// 	return nil, err
	// } else {
	// 	ent = v
	// }

	// // invalid agent
	// if ent == nil && pack != "" {
	// 	return nil, fmt.Errorf("agent not found: %s", pack)
	// }

	// if ent == nil {
	// 	// super agent auto selection
	// 	pack = "agent"
	// 	owner = user
	// } else {
	// 	pack = ent.Name
	// 	owner = ent.Owner
	// }

	// access to models/tools is implicitly granted if user has permission to run the agent
	// agent config
	ac, err := loadAgentsConfig(auth, owner, pack)
	if err != nil {
		return nil, err
	}

	// log.Debugf("creating agent. owner: %s agent: %s %s", owner, pack, sub)

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
