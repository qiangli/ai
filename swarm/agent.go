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

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
)

const defaultMaxTurns = 32
const defaultMaxTime = 3600

var agentRegistry map[string]*api.AgentsConfig

func initDefaultAgents(app *api.AppConfig) error {
	agentRegistry = make(map[string]*api.AgentsConfig)

	config, err := LoadDefaultAgentConfig(app)
	if err != nil {
		log.Errorf("failed to load default tool config: %v", err)
		return err
	}

	for name, v := range config {
		log.Debugf("Registering agent: %s with %d configurations", name, len(v.Agents))
		if len(v.Agents) == 0 {
			log.Infof("No agent configurations found for: %s", name)
			continue
		}
		// Register the agent configurations
		for _, agent := range v.Agents {
			if _, exists := agentRegistry[agent.Name]; exists {
				log.Infof("Duplicate agent name found: %s, skipping registration", agent.Name)
				continue
			}
			// Register the agents configuration
			agentRegistry[agent.Name] = v
			log.Debugf("Registered agent: %s", agent.Name)
			if v.MaxTurns == 0 {
				v.MaxTurns = defaultMaxTurns
			}
			if v.MaxTime == 0 {
				v.MaxTime = defaultMaxTime
			}
		}
	}
	log.Infof("Initialized %d agent configurations", len(agentRegistry))

	return nil
}

type Agent struct {
	// The name of the agent.
	Name string

	Display string

	// The model to be used by the agent
	Model *api.Model

	// The role of the agent. default is "system"
	Role string

	// Instructions for the agent, can be a string or a callable returning a string
	Instruction string

	RawInput *api.UserInput

	// Vars *Vars

	// Functions that the agent can call
	Tools []*api.ToolFunc

	Entrypoint api.Entrypoint

	Dependencies []*Agent

	// advices
	BeforeAdvice api.Advice
	AfterAdvice  api.Advice
	AroundAdvice api.Advice

	//
	MaxTurns int
	MaxTime  int

	//
	// sw *Swarm
	Vars *api.Vars
}

//go:embed agent/resource
var resourceAgents embed.FS

func LoadDefaultAgentConfig(app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
	const base = "agent/resource"
	dirs, err := resourceAgents.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent resource directory: %w", err)
	}

	var agents = make(map[string]*api.AgentsConfig)

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		name := filepath.Join(base, dir.Name(), "agent.yaml")
		// check file exists
		f, err := resourceAgents.ReadFile(name)
		// check if the file exists

		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read agent file %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			log.Infof("agent file is empty %s", name)
			continue
		}
		agent, err := loadAgentsData(app, [][]byte{f})
		if err != nil {
			return nil, fmt.Errorf("failed to load agent data from %s: %w", dir.Name(), err)
		}
		if agent == nil {
			log.Infof("no agent found in %s", dir.Name())
			continue
		}
		// TODO read instruction file if exists
		// use the name of the directory as the default agent name if not specified
		if agent.Name == "" {
			agent.Name = dir.Name()
		}
		if _, exists := agents[agent.Name]; exists {
			// if the agent name already exists, log a warning and skip
			log.Infof("duplicate agent name found: %s in %s, skipping", agent.Name, dir.Name())
			continue
		}
		agents[agent.Name] = agent
	}

	return agents, nil
}

// loadAgentsConfig loads the agent configuration from the provided YAML data.
func loadAgentsData(app *api.AppConfig, data [][]byte) (*api.AgentsConfig, error) {
	merged := &api.AgentsConfig{}

	for _, v := range data {
		cfg := &api.AgentsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}
		if cfg.Internal && !app.Internal {
			// skip internal agents if the app is not internal
			log.Debugf("skip internal agent: %s", cfg.Name)
			continue
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

// func loadAgentsData(data [][]byte) (*api.AgentsConfig, error) {
// 	merged, err := loadAgentsConfig(data)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// merged.ResourceMap = r.ResourceMap
// 	// merged.TemplateFuncMap = r.TemplateFuncMap

// 	// // TODO per agent?
// 	// merged.AdviceMap = r.AdviceMap
// 	// merged.EntrypointMap = r.EntrypointMap

// 	// r.Config = merged

// 	return merged, nil
// }

// func (r *Agent) Vars() *api.Vars {
// 	return r.sw.Vars
// }

func CreateAgent(vars *api.Vars, name, command string, input *api.UserInput) (*Agent, error) {
	config, err := LoadAgents(vars.Config, name, input)
	if err != nil {
		return nil, err
	}

	// config := r.Config
	adviceMap := vars.AdviceMap

	getTools := func(toolType string) ([]*api.ToolFunc, error) {
		var list []*api.ToolFunc
		for _, v := range vars.ToolRegistry {
			if toolType == "*" || v.Type == toolType {
				list = append(list, v)
			}
		}
		return list, nil
	}

	getTool := func(s string) (*api.ToolFunc, error) {
		for _, v := range vars.ToolRegistry {
			if v.Name == s {
				return v, nil
			}
		}
		return nil, fmt.Errorf("no such tool: %s", s)
	}

	getAgentConfig := func(n, c string) (*api.AgentConfig, error) {
		// check for more specific agent first
		if c != "" {
			ap := fmt.Sprintf("%s/%s", n, c)
			for _, a := range config.Agents {
				if a.Name == ap {
					return &a, nil
				}
			}
		}
		for _, a := range config.Agents {
			if a.Name == n {
				return &a, nil
			}
		}
		return nil, fmt.Errorf("no such agent: %s / %s", n, c)
	}

	newAgent := func(ac *api.AgentConfig, vars *api.Vars) (*Agent, error) {
		agent := Agent{
			Name:        ac.Name,
			Display:     ac.Display,
			Role:        ac.Instruction.Role,
			Instruction: ac.Instruction.Content,
			// Vars:        vars,
			RawInput: input,
			MaxTurns: config.MaxTurns,
			MaxTime:  config.MaxTime,
		}

		// override from command line flags
		// if r.AppConfig.Role != "" {
		// 	agent.Role = r.AppConfig.Role
		// }
		// if r.AppConfig.Prompt != "" {
		// 	agent.Instruction = r.AppConfig.Prompt
		// }
		// if r.AppConfig.MaxTurns != 0 {
		// 	agent.MaxTurns = r.AppConfig.MaxTurns
		// }
		// if r.AppConfig.MaxTime != 0 {
		// 	agent.MaxTime = r.AppConfig.MaxTime
		// }

		level := toModelLevel(ac.Model)
		model, ok := vars.Models[level]
		if !ok {
			return nil, fmt.Errorf("no such model: %s", ac.Model)
		}
		agent.Model = model

		// tools
		funcMap := make(map[string]*api.ToolFunc)
		for _, f := range ac.Functions {
			if f == "*" {
				funcs, err := getTools("*")
				if err != nil {
					return nil, err
				}
				for _, fn := range funcs {
					funcMap[fn.ID()] = fn
				}
				continue
			}
			// type:*
			if strings.Contains(f, ":") {
				parts := strings.SplitN(f, ":", 2)
				if len(parts) > 0 {
					funcs, err := getTools(parts[0])
					if err != nil {
						return nil, err
					}
					for _, fn := range funcs {
						funcMap[fn.ID()] = fn
					}
					continue
				}
			}

			// builtin functions
			fn, err := getTool(f)
			if err != nil {
				return nil, err
			}
			funcMap[fn.ID()] = fn
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
			if v.Type == ToolTypeAgent && v.Kit == agentName {
				continue
			}
			funcs = append(funcs, v)
		}
		agent.Tools = funcs

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
		if ac.Entrypoint != "" {
			if ep, ok := vars.EntrypointMap[ac.Entrypoint]; ok {
				agent.Entrypoint = ep
			} else {
				return nil, fmt.Errorf("no such entrypoint: %s", ac.Entrypoint)
			}
		}

		return &agent, nil
	}

	creator := func(vars *api.Vars, name, command string) (*Agent, error) {
		agentCfg, err := getAgentConfig(name, command)
		if err != nil {
			return nil, err
		}
		var deps []*Agent

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
		agent.Vars = vars
		return agent, nil
	}

	return creator(vars, name, command)
}

func LoadAgents(app *api.AppConfig, name string, input *api.UserInput) (*api.AgentsConfig, error) {
	if config, exists := agentRegistry[name]; exists {
		return config, nil
	}
	return nil, fmt.Errorf("no agent configurations found for: %s", name)
}

func (r *Agent) Serve(req *api.Request, resp *api.Response) error {
	log.Debugf("run agent: %s\n", r.Name)

	ctx := req.Context()

	// dependencies
	if len(r.Dependencies) > 0 {
		for _, dep := range r.Dependencies {
			depReq := &api.Request{
				Agent:    dep.Name,
				RawInput: req.RawInput,
				Message:  req.Message,
			}
			depResp := &api.Response{}
			sw := New(r.Vars)
			if err := sw.Run(depReq, depResp); err != nil {
				return err
			}
			log.Debugf("run dependency: %v %+v\n", dep.Display, depResp)
		}
	}

	// advices
	noop := func(vars *api.Vars, _ *api.Request, _ *api.Response, _ api.Advice) error {
		return nil
	}
	if r.BeforeAdvice != nil {
		if err := r.BeforeAdvice(r.Vars, req, resp, noop); err != nil {
			return err
		}
	}
	if r.AroundAdvice != nil {
		next := func(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
			return r.runLoop(ctx, req, resp)
		}
		if err := r.AroundAdvice(r.Vars, req, resp, next); err != nil {
			return err
		}
	} else {
		if err := r.runLoop(ctx, req, resp); err != nil {
			return err
		}
	}
	if r.AfterAdvice != nil {
		if err := r.AfterAdvice(r.Vars, req, resp, noop); err != nil {
			return err
		}
	}

	return nil
}

func (r *Agent) runLoop(ctx context.Context, req *api.Request, resp *api.Response) error {
	// "resource:" prefix is used to refer to a resource
	// "vars:" prefix is used to refer to a variable
	apply := func(s string, vars *api.Vars) (string, error) {
		if strings.HasPrefix(s, "resource:") {
			v, ok := vars.ResourceMap[s[9:]]
			if !ok {
				return "", fmt.Errorf("no such resource: %s", s[9:])
			}
			return applyTemplate(v, vars, vars.TemplateFuncMap)
		}
		if strings.HasPrefix(s, "vars:") {
			v := vars.GetString(s[5:])
			return v, nil
		}
		return s, nil
	}

	var history []*api.Message

	// system role prompt as first message
	if r.Instruction != "" {
		// update the request instruction
		content, err := apply(r.Instruction, r.Vars)
		if err != nil {
			return err
		}

		role := r.Role
		if role == "" {
			role = api.RoleSystem
		}
		history = append(history, &api.Message{
			Role:    role,
			Content: content,
			Sender:  r.Name,
		})
	}
	// FIXME: this is confusing LLM?
	// history = append(history, r.sw.History...)

	if req.Message == nil {
		req.Message = &api.Message{
			Role:    api.RoleUser,
			Content: req.RawInput.Query(),
			Sender:  r.Name,
		}
	}
	history = append(history, req.Message)

	initLen := len(history)
	agentRole := r.Role
	if agentRole == "" {
		agentRole = api.RoleAssistant
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		return CallTool(ctx, r.Vars, name, args)
	}

	result, err := llm.Send(ctx, &api.LLMRequest{
		Agent:     r.Name,
		ModelType: r.Model.Type,
		BaseUrl:   r.Model.BaseUrl,
		ApiKey:    r.Model.ApiKey,
		Model:     r.Model.Name,
		Messages:  history,
		MaxTurns:  r.MaxTurns,
		RunTool:   runTool,
		Tools:     r.Tools,
		//
		ImageQuality: req.ImageQuality,
		ImageSize:    req.ImageSize,
		ImageStyle:   req.ImageStyle,
	})
	if err != nil {
		return err
	}

	if result.Result == nil || result.Result.State != api.StateTransfer {
		message := api.Message{
			ContentType: result.ContentType,
			Role:        result.Role,
			Content:     result.Content,
			Sender:      r.Name,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	resp.Agent = &api.Agent{
		Name:    r.Name,
		Display: r.Display,
	}
	resp.Result = result.Result

	// r.Vars.History = history
	r.Vars.History = append(r.Vars.History, history...)
	return nil
}
