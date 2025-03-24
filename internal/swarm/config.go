package swarm

import (
	"fmt"
	"strings"
	"text/template"
)

type UserConfig struct {
	Name    string `yaml:"name"`
	Display string `yaml:"display"`
}

type AgentsConfig struct {
	User      UserConfig       `yaml:"user"`
	Agents    []AgentConfig    `yaml:"agents"`
	Functions []FunctionConfig `yaml:"functions"`
	Models    []ModelConfig    `yaml:"models"`

	MaxTurns int `yaml:"maxTurns"`
	MaxTime  int `yaml:"maxTime"`

	ResourceMap     map[string]string     `yaml:"-"`
	AdviceMap       map[string]Advice     `yaml:"-"`
	EntrypointMap   map[string]Entrypoint `yaml:"-"`
	TemplateFuncMap template.FuncMap      `yaml:"-"`
}

type AgentConfig struct {
	Name        string `yaml:"name"`
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	//
	Instruction PromptConfig `yaml:"instruction"`

	Model string `yaml:"model"`

	Entrypoint string `yaml:"entrypoint"`

	Functions []string `yaml:"functions"`

	Dependencies []string `yaml:"dependencies"`

	Advices AdviceConfig `yaml:"advices"`
}

type PromptConfig struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

type FunctionConfig struct {
	Label   string `yaml:"label"`
	Service string `yaml:"service"`

	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`
}

type ModelConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Model       string `yaml:"model"`
	BaseUrl     string `yaml:"baseUrl"`
	ApiKey      string `yaml:"apiKey"`
	External    bool   `yaml:"external"`
}

type AdviceConfig struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Around string `yaml:"around"`
}

func (r *Swarm) Create(name, command string, input *UserInput) (*Agent, error) {
	if err := r.Load(name, input); err != nil {
		return nil, err
	}

	config := r.Config
	adviceMap := config.AdviceMap

	getTools := func(toolType string) ([]*ToolFunc, error) {
		var list []*ToolFunc
		for _, v := range r.Vars.ToolRegistry {
			if toolType == "*" || v.Type == toolType {
				list = append(list, v)
			}
		}
		return list, nil
	}

	getTool := func(s string) (*ToolFunc, error) {
		for _, v := range r.Vars.ToolRegistry {
			if v.Name == s {
				return v, nil
			}
		}
		return nil, fmt.Errorf("no such tool: %s", s)
	}

	getAgentConfig := func(n, c string) (*AgentConfig, error) {
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

	newAgent := func(ac *AgentConfig, vars *Vars) (*Agent, error) {
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
		if r.AppConfig.Role != "" {
			agent.Role = r.AppConfig.Role
		}
		if r.AppConfig.Prompt != "" {
			agent.Instruction = r.AppConfig.Prompt
		}
		if r.AppConfig.MaxTurns != 0 {
			agent.MaxTurns = r.AppConfig.MaxTurns
		}
		if r.AppConfig.MaxTime != 0 {
			agent.MaxTime = r.AppConfig.MaxTime
		}

		level := toModelLevel(ac.Model)
		model, ok := vars.Models[level]
		if !ok {
			return nil, fmt.Errorf("no such model: %s", ac.Model)
		}
		agent.Model = model

		// tools
		funcMap := make(map[string]*ToolFunc)
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
		var funcs []*ToolFunc
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
			if ep, ok := config.EntrypointMap[ac.Entrypoint]; ok {
				agent.Entrypoint = ep
			} else {
				return nil, fmt.Errorf("no such entrypoint: %s", ac.Entrypoint)
			}
		}

		return &agent, nil
	}

	creator := func(name, command string, vars *Vars) (*Agent, error) {
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

		return agent, nil
	}

	return creator(name, command, r.Vars)
}
