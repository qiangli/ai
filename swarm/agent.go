package swarm

import (
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

const defaultMaxTurns = 15
const defaultMaxTime = 3600

func initAgents(app *api.AppConfig) (func(string) (*api.AgentsConfig, error), error) {
	agents, err := ListAgents(app)
	if err != nil {
		return nil, err
	}
	return func(name string) (*api.AgentsConfig, error) {
		if config, ok := agents[name]; ok {
			return config, nil
		}
		return nil, fmt.Errorf("no agent configurations found for: %s", name)
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
			log.Debugf("No agent configurations found for: %s\n", name)
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
		log.Debugf("No agent configurations found in default agents\n")
		return nil, fmt.Errorf("no agent configurations found in default agents")
	}
	log.Debugf("Initialized %d agent configurations\n", len(agentRegistry))
	return agentRegistry, nil
}

func LoadAgentsConfig(app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	var groups = make(map[string]*api.AgentsConfig)
	// default
	if err := LoadResourceAgentsConfig(app, groups); err != nil {
		return nil, err
	}

	// external/custom
	if err := LoadFileAgentsConfig(app.Base, groups); err != nil {
		log.Errorf("failed to load custom agents: %v\n", err)
	}

	// web
	if err := LoadWebAgentsConfig(app, groups); err != nil {
		log.Errorf("failed load agents from web resources: %v\n", err)
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
			return fmt.Errorf("failed to read agent file %s: %w", dir.Name(), err)
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

func LoadResourceAgentsConfig(app *api.AppConfig, groups map[string]*api.AgentsConfig) error {
	rs := &ResourceStore{
		Base: "resource",
	}
	return LoadAgentsAsset(rs, "agents", groups)
}

func LoadFileAgentsConfig(base string, groups map[string]*api.AgentsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	fs := &FileStore{
		Base: abs,
	}
	return LoadAgentsAsset(fs, "agents", groups)
}

func LoadWebAgentsConfig(app *api.AppConfig, groups map[string]*api.AgentsConfig) error {
	if app.AgentResource == nil {
		return nil
	}
	for _, base := range app.AgentResource.Bases {
		ws := &WebStore{
			Base: base,
		}
		if err := LoadAgentsAsset(ws, "agents", groups); err != nil {
			log.Errorf("failed to load config. base: %s error: %v\n", base, err)
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
		// if cfg.Internal && !app.Internal {
		// 	// skip internal agents if the app is not internal
		// 	log.Debugf("skip internal agent: %s\n", cfg.Name)
		// 	continue
		// }

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

	//
	adviceMap := vars.AdviceMap

	// TODO - check if the tool type is enabled
	// by default all tools are enabled
	// except mcp which is enabled only if the mcp server root is set
	// isEnabled := func(toolType string) bool {
	// 	return toolType != "mcp" || vars.Config.McpServerRoot != ""
	// }

	// getTools := func(toolType string, kit string) ([]*api.ToolFunc, error) {
	// 	var list []*api.ToolFunc
	// 	for _, v := range vars.ToolRegistry {
	// 		if toolType == "*" || toolType == "" || v.Type == toolType {
	// 			if kit == "*" || kit == "" || v.Kit == kit {
	// 				list = append(list, v)
	// 			}
	// 		}
	// 	}
	// 	if len(list) == 0 {
	// 		if isEnabled(toolType) {
	// 			return nil, fmt.Errorf("no such tool: %s / %s", toolType, kit)
	// 		}
	// 	}
	// 	return list, nil
	// }

	// getTool := func(s string) (*api.ToolFunc, error) {
	// 	for _, v := range vars.ToolRegistry {
	// 		if v.Name == s {
	// 			return v, nil
	// 		}
	// 	}
	// 	return nil, fmt.Errorf("no such tool: %s", s)
	// }

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
			Name:    ac.Name,
			Display: ac.Display,
			//
			Config: ac,
			//
			RawInput: input,
			MaxTurns: config.MaxTurns,
			MaxTime:  config.MaxTime,
		}

		model, err := vars.Config.ModelLoader(ac.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to load model %q: %v", ac.Model, err)
		}
		agent.Model = model

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

			// // function by name
			// fn, err := getTool(f)
			// if err != nil {
			// 	return nil, err
			// }
			// if fn != nil {
			// 	funcMap[fn.ID()] = fn
			// }
		}

		// FIXME
		// 1. better handle this to avoid agent calling self as tools
		// filter out all agent tools from the agent itself
		// 2. handle namespace to avoid collision of tool names
		agentName := ac.Name
		if strings.Contains(agentName, "/") {
			parts := strings.SplitN(agentName, "/", 2)
			agentName = parts[0]
		}
		var funcs []*api.ToolFunc
		for _, v := range funcMap {
			if v.Type == ToolTypeAgent && v.Name == agentName {
				continue
			}
			funcs = append(funcs, v)
		}
		agent.Tools = funcs

		if ac.Advices != nil {
			if ac.Advices.Before != "" {
				if ad, ok := adviceMap[ac.Advices.Before]; ok {
					agent.BeforeAdvice = ad
				} else {
					return nil, fmt.Errorf("no such advice: %s", ac.Advices.Before)
				}
			}
			if ac.Advices.After != "" {
				if ad, ok := adviceMap[ac.Advices.After]; ok {
					agent.AfterAdvice = ad
				} else {
					return nil, fmt.Errorf("no such advice: %s", ac.Advices.After)
				}
			}
			if ac.Advices.Around != "" {
				if ad, ok := adviceMap[ac.Advices.Around]; ok {
					agent.AroundAdvice = ad
				} else {
					return nil, fmt.Errorf("no such advice: %s", ac.Advices.Around)
				}
			}
		}
		if ac.Entrypoint != "" {
			if ep, ok := vars.EntrypointMap[ac.Entrypoint]; ok {
				agent.Entrypoint = ep
			} else {
				return nil, fmt.Errorf("no such entrypoint: %s", ac.Entrypoint)
			}
		}

		return &agent, nil
	}

	creator := func(vars *api.Vars, name, command string) (*api.Agent, error) {
		agentCfg, err := getAgentConfig(name, command)
		if err != nil {
			return nil, err
		}
		var deps []*api.Agent

		if len(agentCfg.Dependencies) > 0 {
			for _, dep := range agentCfg.Dependencies {
				depCfg, err := getAgentConfig(dep, "")
				if err != nil {
					return nil, err
				}
				agent, err := newAgent(depCfg, vars)
				if err != nil {
					return nil, err
				}
				deps = append(deps, agent)
			}
		}
		agent, err := newAgent(agentCfg, vars)
		if err != nil {
			return nil, err
		}
		agent.Dependencies = deps
		// agent.Vars = vars
		return agent, nil
	}

	return creator(vars, name, command)
}

// func LoadAgents(app *api.AppConfig, name string) (*api.AgentsConfig, error) {
// 	if config, ok := app.AgentRegistry[name]; ok {
// 		return config, nil
// 	}
// 	return nil, fmt.Errorf("no agent configurations found for: %s", name)
// }
