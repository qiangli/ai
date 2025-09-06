package swarm

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/api/model"
)

const defaultMaxTurns = 8
const defaultMaxTime = 300 // 5 min

func initAgents(app *api.AppConfig) (func(string) (*api.AgentsConfig, error), error) {
	agents, err := ListAgents(app)
	if err != nil {
		return nil, err
	}
	return func(name string) (*api.AgentsConfig, error) {
		if config, ok := agents[name]; ok {
			return config, nil
		}
		return nil, fmt.Errorf("agent not found: %s", name)
	}, nil
}

func AgentLister(app *api.AppConfig) (func() map[string]*api.AgentsConfig, error) {
	agents, err := ListAgents(app)
	if err != nil {
		return nil, err
	}
	return func() map[string]*api.AgentsConfig {
		return agents
	}, nil
}

func ListAgents(app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	var agentRegistry = make(map[string]*api.AgentsConfig)

	config, err := LoadAgentsConfig(app)
	if err != nil {
		log.Errorf("failed to load default tool config: %v\n", err)
		return nil, err
	}

	for name, v := range config {
		log.Debugf("Registering agent: %s with %d configurations\n", name, len(v.Agents))
		if len(v.Agents) == 0 {
			log.Debugf("No agents found in config: %s\n", name)
			continue
		}
		// Register the agent configurations
		for _, agent := range v.Agents {
			if _, exists := agentRegistry[agent.Name]; exists {
				log.Debugf("Duplicate agent name found: %s, skipping registration\n", agent.Name)
				continue
			}
			// Register the agents configuration
			agentRegistry[agent.Name] = v
			log.Debugf("Registered agent: %s\n", agent.Name)
			if v.MaxTurns == 0 {
				v.MaxTurns = defaultMaxTurns
			}
			if v.MaxTime == 0 {
				v.MaxTime = defaultMaxTime
			}
		}
	}

	if len(agentRegistry) == 0 {
		return nil, fmt.Errorf("no agent configurations found")
	}
	log.Debugf("Initialized %d agent configurations\n", len(agentRegistry))
	return agentRegistry, nil
}

func LoadAgentsConfig(app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	var groups = make(map[string]*api.AgentsConfig)
	// default
	if err := LoadResourceAgentsConfig(resourceFS, groups); err != nil {
		return nil, err
	}

	// external/custom
	if err := LoadFileAgentsConfig(app.Base, groups); err != nil {
		log.Errorf("failed to load custom agents: %v\n", err)
	}

	// web
	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebAgentsConfig(app.AgentResource.Resources, groups); err != nil {
			log.Errorf("failed load agents from web resources: %v\n", err)
		}
	}

	return groups, nil
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
		base := filepath.Join(root, dir.Name())
		name := filepath.Join(base, "agent.yaml")
		f, err := as.ReadFile(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read agent asset %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			log.Debugf("agent file is empty %s\n", name)
			continue
		}
		group, err := LoadAgentsData([][]byte{f})
		if err != nil {
			return fmt.Errorf("failed to load agent data from %s: %w", dir.Name(), err)
		}
		if group == nil {
			log.Debugf("no agent found in %s\n", dir.Name())
			continue
		}
		// group.BaseDir = base
		// use the name of the directory as the group name if not specified
		if group.Name == "" {
			group.Name = dir.Name()
		}
		if _, exists := groups[group.Name]; exists {
			log.Debugf("duplicate agent name found: %s in %s, skipping\n", group.Name, dir.Name())
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

func LoadResourceAgentsConfig(fs embed.FS, groups map[string]*api.AgentsConfig) error {
	rs := &ResourceStore{
		FS:   fs,
		Base: "resource",
	}
	return LoadAgentsAsset(rs, "agents", groups)
}

func LoadFileAgentsConfig(base string, groups map[string]*api.AgentsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	// check if abs exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.Debugf("path does not exist: %s\n", abs)
		return nil
	}

	fs := &FileStore{
		Base: abs,
	}
	return LoadAgentsAsset(fs, "agents", groups)
}

func LoadWebAgentsConfig(resources []*api.Resource, groups map[string]*api.AgentsConfig) error {
	for _, v := range resources {
		ws := &WebStore{
			Base:  v.Base,
			Token: v.Token,
		}
		if err := LoadAgentsAsset(ws, "agents", groups); err != nil {
			log.Errorf("*** failed to load config. base: %s error: %v\n", v.Base, err)
		}
	}
	return nil
}

// LoadAgentsConfig loads the agent configuration from the provided YAML data.
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

func CreateAgent(vars *api.Vars, name, command string, input *api.UserInput) (*api.Agent, error) {
	// config, err := LoadAgents(vars.Config, name)
	config, err := vars.Config.AgentLoader(name)
	if err != nil {
		return nil, err
	}

	findAgentConfig := func(n, c string) (*api.AgentConfig, error) {
		// check for more specific agent first
		if c != "" {
			ap := fmt.Sprintf("%s/%s", n, c)
			for _, a := range config.Agents {
				if a.Name == ap {
					return a, nil
				}
			}
		}
		for _, a := range config.Agents {
			if a.Name == n {
				return a, nil
			}
		}
		return nil, fmt.Errorf("no such agent: %s / %s", n, c)
	}

	getAgentConfig := func(n, c string) (*api.AgentConfig, error) {
		a, err := findAgentConfig(n, c)
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
				log.Debugf("Loaded instruction from file %q for agent %q\n", resource, a.Name)
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
				log.Debugf("Loaded instruction from resource %q for agent %q\n", resource, a.Name)
			}
		}
		return a, nil
	}

	newAgent := func(ac *api.AgentConfig, vars *api.Vars) (*api.Agent, error) {
		agent := api.Agent{
			Adapter: ac.Adapter,
			Name:    ac.Name,
			Display: ac.Display,
			//
			Model: ac.Model,
			//
			Config: ac,
			//
			RawInput: input,
			MaxTurns: config.MaxTurns,
			MaxTime:  config.MaxTime,
			//
			Dependencies: ac.Dependencies,
		}

		// model, err := vars.Config.ModelLoader(ac.Model)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to load model %q: %v", ac.Model, err)
		// }
		// agent.Model = model

		// tools
		funcMap := make(map[string]*api.ToolFunc)
		for _, f := range ac.Functions {
			// all
			if f == "*" || f == "*:" || f == "*:*" {
				funcs, err := vars.Config.ToolLoader("*:*")
				if err != nil {
					return nil, err
				}
				for _, fn := range funcs {
					funcMap[fn.ID()] = fn
				}
				continue
			}
			// kit:*
			if strings.Contains(f, ":") {
				parts := strings.SplitN(f, ":", 2)
				if len(parts) > 0 {
					kit := "*"
					if len(parts) > 1 && len(parts[1]) > 0 {
						kit = parts[1]
					}
					funcs, err := vars.Config.ToolLoader(fmt.Sprintf("%s:%s", parts[0], kit))
					if err != nil {
						return nil, err
					}
					for _, fn := range funcs {
						funcMap[fn.ID()] = fn
					}
					continue
				}
			}
		}

		// FIXME
		// 1. better handle this to avoid agent calling self as tools
		// filter out all agent tools from the agent itself
		// 2. handle namespace to avoid collision of tool names
		// agentName := ac.Name
		// if strings.Contains(agentName, "/") {
		// 	parts := strings.SplitN(agentName, "/", 2)
		// 	agentName = parts[0]
		// }

		var funcs []*api.ToolFunc
		for _, v := range funcMap {
			// if v.Type == ToolTypeAgent && v.Name == agentName {
			// 	continue
			// }
			funcs = append(funcs, v)
		}
		agent.Tools = funcs

		if ac.Advices != nil {
			// TODO
			return nil, fmt.Errorf("advice no supported: %+v", ac.Advices)
			// if ac.Advices.Before != "" {
			// 	if ad, ok := adviceMap[ac.Advices.Before]; ok {
			// 		agent.BeforeAdvice = ad
			// 	} else {
			// 		return nil, fmt.Errorf("no such advice: %s", ac.Advices.Before)
			// 	}
			// }
			// if ac.Advices.After != "" {
			// 	if ad, ok := adviceMap[ac.Advices.After]; ok {
			// 		agent.AfterAdvice = ad
			// 	} else {
			// 		return nil, fmt.Errorf("no such advice: %s", ac.Advices.After)
			// 	}
			// }
			// if ac.Advices.Around != "" {
			// 	if ad, ok := adviceMap[ac.Advices.Around]; ok {
			// 		agent.AroundAdvice = ad
			// 	} else {
			// 		return nil, fmt.Errorf("no such advice: %s", ac.Advices.Around)
			// 	}
			// }
		}
		if ac.Entrypoint != "" {
			return nil, fmt.Errorf("entrypoint not supported: %s", ac.Entrypoint)
			// if ep, ok := vars.EntrypointMap[ac.Entrypoint]; ok {
			// 	agent.Entrypoint = ep
			// } else {
			// 	return nil, fmt.Errorf("no such entrypoint: %s", ac.Entrypoint)
			// }
		}

		return &agent, nil
	}

	creator := func(vars *api.Vars, name, command string) (*api.Agent, error) {
		agentCfg, err := getAgentConfig(name, command)
		if err != nil {
			return nil, err
		}

		agent, err := newAgent(agentCfg, vars)
		if err != nil {
			return nil, err
		}

		return agent, nil
	}

	return creator(vars, name, command)
}
