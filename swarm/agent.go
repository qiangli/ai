package swarm

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

var getApiKey func(string) string

func init() {
	// save the env before they are cleared during runtime
	var env = make(map[string]string)
	env["openai"] = os.Getenv("OPENAI_API_KEY")
	env["gemini"] = os.Getenv("GEMINI_API_KEY")
	env["anthropic"] = os.Getenv("ANTHROPIC_API_KEY")
	//
	env["google"] = os.Getenv("GOOGLE_SEARCH_ENGINE_ID") + ":" + os.Getenv("GOOGLE_API_KEY")
	// env["GOOGLE_SEARCH_ENGINE_ID"] = os.Getenv("GOOGLE_SEARCH_ENGINE_ID")
	env["brave"] = os.Getenv("BRAVE_API_KEY")

	getApiKey = func(provider string) string {
		return env[provider]
	}
}

func provideApiKey(key string) func() (string, error) {
	return func() (string, error) {
		ak := getApiKey(key)
		if ak != "" {
			return ak, nil
		}
		return "", fmt.Errorf("api key not found: %s", key)
	}
}

// max hard upper limits
const maxTurnsLimit = 100
const maxTimeLimit = 600 // 10 min

const defaultMaxTurns = 8
const defaultMaxTime = 180 // 3 min

func NewAgentCreator() api.AgentCreator {
	return func(vars *api.Vars, req *api.Request) (*api.Agent, error) {
		return CreateAgent(req.Context(), vars, req.Agent, req.RawInput)
	}
}

func initAgents(ctx context.Context, app *api.AppConfig) (func(string) (*api.AgentsConfig, error), error) {
	agents, err := ListAgents(ctx, app)
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

func ListAgents(ctx context.Context, app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	var agents = make(map[string]*api.AgentsConfig)

	config, err := LoadAgentsConfig(ctx, app)
	if err != nil {
		log.GetLogger(ctx).Error("failed to load default tool config: %v\n", err)
		return nil, err
	}

	for name, v := range config {
		log.GetLogger(ctx).Debug("Registering agent: %s with %d configurations\n", name, len(v.Agents))
		if len(v.Agents) == 0 {
			log.GetLogger(ctx).Debug("No agents found in config: %s\n", name)
			continue
		}
		// Register the agent configurations
		for _, agent := range v.Agents {
			if _, exists := agents[agent.Name]; exists {
				log.GetLogger(ctx).Debug("Duplicate agent name found: %s, skipping registration\n", agent.Name)
				continue
			}
			// Register the agents configuration
			agents[agent.Name] = v
			log.GetLogger(ctx).Debug("Registered agent: %s\n", agent.Name)
			if v.MaxTurns == 0 {
				v.MaxTurns = defaultMaxTurns
			}
			if v.MaxTime == 0 {
				v.MaxTime = defaultMaxTime
			}
			// upper limit
			v.MaxTurns = min(v.MaxTurns, maxTurnsLimit)
			v.MaxTime = min(v.MaxTime, maxTimeLimit)
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agent configurations found")
	}
	log.GetLogger(ctx).Debug("Initialized %d agent configurations\n", len(agents))
	return agents, nil
}

func LoadAgentsConfig(ctx context.Context, app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	var groups = make(map[string]*api.AgentsConfig)

	// web
	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebAgentsConfig(ctx, app.AgentResource.Resources, groups); err != nil {
			log.GetLogger(ctx).Error("failed load agents from web resources: %v\n", err)
		}
	}

	// external/custom
	if err := LoadFileAgentsConfig(ctx, app.Base, groups); err != nil {
		log.GetLogger(ctx).Error("failed to load custom agents: %v\n", err)
	}

	// default
	if err := LoadResourceAgentsConfig(ctx, resourceFS, groups); err != nil {
		return nil, err
	}

	return groups, nil
}

func LoadAgentsAsset(ctx context.Context, as api.AssetStore, root string, groups map[string]*api.AgentsConfig) error {
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
			log.GetLogger(ctx).Debug("agent file is empty %s\n", name)
			continue
		}
		group, err := LoadAgentsData([][]byte{f})
		if err != nil {
			return fmt.Errorf("failed to load agent data from %s: %w", dir.Name(), err)
		}
		if group == nil {
			log.GetLogger(ctx).Debug("no agent found in %s\n", dir.Name())
			continue
		}
		// group.BaseDir = base
		// use the name of the directory as the group name if not specified
		if group.Name == "" {
			group.Name = dir.Name()
		}
		if _, exists := groups[group.Name]; exists {
			log.GetLogger(ctx).Debug("duplicate agent name found: %s in %s, skipping\n", group.Name, dir.Name())
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

func LoadResourceAgentsConfig(ctx context.Context, fs embed.FS, groups map[string]*api.AgentsConfig) error {
	rs := &ResourceStore{
		FS:   fs,
		Base: "resource",
	}
	return LoadAgentsAsset(ctx, rs, "agents", groups)
}

func LoadFileAgentsConfig(ctx context.Context, base string, groups map[string]*api.AgentsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	// check if abs exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.GetLogger(ctx).Debug("path does not exist: %s\n", abs)
		return nil
	}

	fs := &FileStore{
		Base: abs,
	}
	return LoadAgentsAsset(ctx, fs, "agents", groups)
}

func LoadWebAgentsConfig(ctx context.Context, resources []*api.Resource, groups map[string]*api.AgentsConfig) error {
	for _, v := range resources {
		ws := &WebStore{
			Base:  v.Base,
			Token: v.Token,
		}
		if err := LoadAgentsAsset(ctx, ws, "agents", groups); err != nil {
			log.GetLogger(ctx).Error("*** failed to load config. base: %s error: %v\n", v.Base, err)
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

	// fill defaults
	for _, v := range merged.Agents {
		// model alias
		if v.Model == "" {
			v.Model = merged.Model
		}
	}

	return merged, nil
}

func CreateAgent(ctx context.Context, vars *api.Vars, name string, input *api.UserInput) (*api.Agent, error) {
	agentLoader, err := initAgents(ctx, vars.Config)
	if err != nil {
		return nil, err
	}
	toolLoader, err := initTools(ctx, vars.Config)
	if err != nil {
		return nil, err
	}
	modelLoader, err := initModels(ctx, vars.Config)
	if err != nil {
		return nil, err
	}

	config, err := agentLoader(name)
	if err != nil {
		return nil, err
	}

	findAgentConfig := func(n string) (*api.AgentConfig, error) {
		// check for more specific agent first
		// if c != "" {
		// 	ap := fmt.Sprintf("%s/%s", n, c)
		// 	for _, a := range config.Agents {
		// 		if a.Name == ap {
		// 			return a, nil
		// 		}
		// 	}
		// }
		for _, a := range config.Agents {
			if a.Name == n {
				return a, nil
			}
		}
		return nil, fmt.Errorf("no such agent: %s", n)
	}

	getAgentConfig := func(n string) (*api.AgentConfig, error) {
		a, err := findAgentConfig(n)
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
				log.GetLogger(ctx).Debug("Loaded instruction from file %q for agent %q\n", resource, a.Name)
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
				log.GetLogger(ctx).Debug("Loaded instruction from resource %q for agent %q\n", resource, a.Name)
			}
		}
		return a, nil
	}

	newAgent := func(ac *api.AgentConfig, vars *api.Vars) (*api.Agent, error) {
		agent := api.Agent{
			Adapter: ac.Adapter,
			//
			Name:        ac.Name,
			Display:     ac.Display,
			Description: ac.Description,
			//
			Instruction: ac.Instruction,
			//
			Config: config,
			//
			RawInput: input,
			MaxTurns: config.MaxTurns,
			MaxTime:  config.MaxTime,
			//
			Dependencies: ac.Dependencies,
		}

		model, err := modelLoader(ac.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to load model %q: %v", ac.Model, err)
		}
		agent.Model = model
		// agent.Model.ApiKey = func() (string, error) {
		// 	ak := getApiKey(model.ApiKey)
		// 	if ak != "" {
		// 		return ak, nil
		// 	}
		// 	return "", fmt.Errorf("api key not found: %s", model.ApiKey)
		// }
		// tools
		funcMap := make(map[string]*api.ToolFunc)
		for _, f := range ac.Functions {
			// all
			if f == "*" || f == "*:" || f == "*:*" {
				funcs, err := toolLoader("*:*")
				if err != nil {
					return nil, err
				}
				for _, fn := range funcs {
					id := fn.ID()
					if id == "" {
						return nil, fmt.Errorf("tool ID is empty. agent: %s", name)
					}
					funcMap[id] = fn
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
					funcs, err := toolLoader(fmt.Sprintf("%s:%s", parts[0], kit))
					if err != nil {
						return nil, err
					}
					for _, fn := range funcs {
						id := fn.ID()
						if id == "" {
							return nil, fmt.Errorf("tool ID is empty agent: %s", name)
						}
						funcMap[id] = fn
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

	creator := func(vars *api.Vars, name string) (*api.Agent, error) {
		agentCfg, err := getAgentConfig(name)
		if err != nil {
			return nil, err
		}

		agent, err := newAgent(agentCfg, vars)
		if err != nil {
			return nil, err
		}

		return agent, nil
	}

	return creator(vars, name)
}
