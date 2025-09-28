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
		log.GetLogger(ctx).Errorf("Failed to load agent config: %v\n", err)
		return nil, err
	}

	for name, v := range config {
		log.GetLogger(ctx).Debugf("Registering agent: %s with %d configurations\n", name, len(v.Agents))
		if len(v.Agents) == 0 {
			log.GetLogger(ctx).Debugf("No agents found in config: %s\n", name)
			continue
		}
		// Register the agent configurations
		for _, agent := range v.Agents {
			if _, exists := agents[agent.Name]; exists {
				log.GetLogger(ctx).Debugf("Duplicate agent name found: %s, skipping registration\n", agent.Name)
				continue
			}
			// Register the agents configuration
			agents[agent.Name] = v
			log.GetLogger(ctx).Debugf("Registered agent: %s\n", agent.Name)

			if v.MaxTurns == 0 {
				v.MaxTurns = defaultMaxTurns
			}
			if v.MaxTime == 0 {
				v.MaxTime = defaultMaxTime
			}
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agent configurations found")
	}
	log.GetLogger(ctx).Debugf("Initialized %d agent configurations\n", len(agents))
	return agents, nil
}

func LoadAgentsConfig(ctx context.Context, app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	var packs = make(map[string]*api.AgentsConfig)

	// web
	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebAgentsConfig(ctx, app.AgentResource.Resources, packs); err != nil {
			log.GetLogger(ctx).Errorf("Failed load agents from web resources: %v\n", err)
		}
	}

	// external/custom
	if err := LoadFileAgentsConfig(ctx, app.Base, packs); err != nil {
		log.GetLogger(ctx).Errorf("Failed to load custom agents: %v\n", err)
	}

	// default
	if err := LoadResourceAgentsConfig(ctx, resourceFS, packs); err != nil {
		return nil, err
	}

	return packs, nil
}

func LoadAgentsAsset(ctx context.Context, as api.AssetStore, root string, packs map[string]*api.AgentsConfig) error {
	dirs, err := as.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
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
			log.GetLogger(ctx).Debugf("Agent file is empty %s\n", name)
			continue
		}
		pack, err := LoadAgentsData([][]byte{f})
		if err != nil {
			return fmt.Errorf("Failed to load agent data from %s: %w", dir.Name(), err)
		}
		if pack == nil {
			log.GetLogger(ctx).Debugf("No agent found in %s\n", dir.Name())
			continue
		}
		// pack.BaseDir = base
		// use the name of the directory as the pack name if not specified
		if pack.Name == "" {
			pack.Name = dir.Name()
		}
		if _, exists := packs[pack.Name]; exists {
			log.GetLogger(ctx).Debugf("Duplicate agent name found: %s in %s, skipping\n", pack.Name, dir.Name())
			continue
		}

		// keep store loader for loading extra resources later
		for _, v := range pack.Agents {
			v.Store = as
			v.BaseDir = base
		}

		packs[pack.Name] = pack
	}

	return nil
}

func LoadResourceAgentsConfig(ctx context.Context, fs embed.FS, packs map[string]*api.AgentsConfig) error {
	rs := &ResourceStore{
		FS:   fs,
		Base: "resource",
	}
	return LoadAgentsAsset(ctx, rs, "agents", packs)
}

func LoadFileAgentsConfig(ctx context.Context, base string, packs map[string]*api.AgentsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	// check if abs exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.GetLogger(ctx).Debugf("Path does not exist: %s\n", abs)
		return nil
	}

	fs := &FileStore{
		Base: abs,
	}
	return LoadAgentsAsset(ctx, fs, "agents", packs)
}

func LoadWebAgentsConfig(ctx context.Context, resources []*api.Resource, packs map[string]*api.AgentsConfig) error {
	for _, v := range resources {
		ws := &WebStore{
			Base:  v.Base,
			Token: v.Token,
		}
		if err := LoadAgentsAsset(ctx, ws, "agents", packs); err != nil {
			log.GetLogger(ctx).Errorf("Failed to load config. base: %s error: %v\n", v.Base, err)
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

	ac, err := agentLoader(name)
	if err != nil {
		return nil, err
	}

	findAgentConfig := func(n string) (*api.AgentConfig, error) {
		for _, a := range ac.Agents {
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
				log.GetLogger(ctx).Debugf("Loaded instruction from file %q for agent %q\n", resource, a.Name)
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
				log.GetLogger(ctx).Debugf("Loaded instruction from resource %q for agent %q\n", resource, a.Name)
			}
		}
		return a, nil
	}

	newAgent := func(c *api.AgentConfig, vars *api.Vars) (*api.Agent, error) {
		agent := api.Agent{
			Adapter: c.Adapter,
			//
			Name:        c.Name,
			Display:     c.Display,
			Description: c.Description,
			//
			Instruction: c.Instruction,
			//
			Config: ac,
			//
			RawInput: input,
			//
			MaxTurns: nzl(vars.Config.MaxTurns, c.MaxTurns, ac.MaxTurns),
			MaxTime:  nzl(vars.Config.MaxTime, c.MaxTime, ac.MaxTime),
			//
			LogLevel: api.Quiet,
			Message:  nvl(vars.Config.Message, c.Message, ac.Message),
			Format:   nvl(vars.Config.Format, c.Format, ac.Format),
			New:      nbl(vars.Config.New, c.New, ac.New),
			//
			Dependencies: c.Dependencies,
		}

		//
		agent.LogLevel = api.ToLogLevel(nvl(vars.Config.LogLevel, c.LogLevel, ac.LogLevel))

		// hard upper limit
		agent.MaxTurns = min(agent.MaxTurns, maxTurnsLimit)
		agent.MaxTime = min(agent.MaxTime, maxTimeLimit)

		model, err := modelLoader(c.Model)
		if err != nil {
			return nil, fmt.Errorf("failed to load model %q: %v", c.Model, err)
		}
		agent.Model = model

		// tools
		funcMap := make(map[string]*api.ToolFunc)
		for _, f := range c.Functions {
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

		var funcs []*api.ToolFunc
		for _, v := range funcMap {
			funcs = append(funcs, v)
		}
		agent.Tools = funcs

		if c.Advices != nil {
			// TODO
			return nil, fmt.Errorf("advice no supported: %+v", c.Advices)
			// if c.Advices.Before != "" {
			// 	if ad, ok := adviceMap[c.Advices.Before]; ok {
			// 		agent.BeforeAdvice = ad
			// 	} else {
			// 		return nil, fmt.Errorf("no such advice: %s", c.Advices.Before)
			// 	}
			// }
			// if c.Advices.After != "" {
			// 	if ad, ok := adviceMap[c.Advices.After]; ok {
			// 		agent.AfterAdvice = ad
			// 	} else {
			// 		return nil, fmt.Errorf("no such advice: %s", c.Advices.After)
			// 	}
			// }
			// if c.Advices.Around != "" {
			// 	if ad, ok := adviceMap[c.Advices.Around]; ok {
			// 		agent.AroundAdvice = ad
			// 	} else {
			// 		return nil, fmt.Errorf("no such advice: %s", c.Advices.Around)
			// 	}
			// }
		}
		if c.Entrypoint != "" {
			return nil, fmt.Errorf("entrypoint not supported: %s", c.Entrypoint)
			// if ep, ok := vars.EntrypointMap[c.Entrypoint]; ok {
			// 	agent.Entrypoint = ep
			// } else {
			// 	return nil, fmt.Errorf("no such entrypoint: %s", c.Entrypoint)
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
