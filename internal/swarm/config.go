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

func (r *Swarm) Create(name string, input *UserInput) (*Agent, error) {
	if err := r.Load(name, input); err != nil {
		return nil, err
	}

	config := r.Config
	adviceMap := config.AdviceMap

	getSystemTools := func() ([]*ToolFunc, error) {
		var list []*ToolFunc
		for _, v := range r.Vars.ToolMap {
			if v.Label == ToolLabelSystem {
				list = append(list, v)
			}
		}
		return list, nil
		// return ListSystemTools()
	}

	getMcpTools := func(s string) ([]*ToolFunc, error) {
		var list []*ToolFunc
		for _, v := range r.Vars.ToolMap {
			if v.Label == ToolLabelMcp {
				list = append(list, v)
			}
		}
		return list, nil

		// parts := strings.SplitN(s, ":", 2)
		// if len(parts) == 2 {
		// 	// mcp:server
		// 	s = parts[1]
		// }
		// var mcpToolMap map[string][]*ToolFunc
		// var err error

		// if s == "" || s == "*" {
		// 	mcpToolMap, err = r.McpServerTool.ListTools()
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// } else {
		// 	tools, err := r.McpServerTool.GetTools(s)
		// 	if err != nil {
		// 		return nil, err
		// 	}
		// 	mcpToolMap = map[string][]*ToolFunc{
		// 		s: tools,
		// 	}
		// }
		// mcpFunctions := []*ToolFunc{}
		// for server, m := range mcpToolMap {
		// 	for _, v := range m {
		// 		// fn := fmt.Sprintf("%s__%s", server, v.Name)
		// 		// mcpFunctions = append(mcpFunctions, &ToolFunc{
		// 		// 	Service:        server,
		// 		// 	Func:           v.Func,
		// 		// 	Description: v.Description,
		// 		// 	Parameters:  v.Parameters,
		// 		// })
		// 		v.Service = server
		// 		mcpFunctions = append(mcpFunctions, v)
		// 	}
		// }
		// return mcpFunctions, nil
	}

	// agent:agent/command
	getAgentTools := func(s string) ([]*ToolFunc, error) {
		var list []*ToolFunc
		for _, v := range r.Vars.ToolMap {
			if v.Label == ToolLabelAgent {
				list = append(list, v)
			}
		}
		return list, nil
		// // skip self
		// if s == name {
		// 	return nil, nil
		// }
		// parts := strings.SplitN(s, ":", 2)
		// if len(parts) == 2 {
		// 	// agent:agent/command
		// 	s = parts[1]
		// }
		// if s == "" || s == "*" {
		// 	agentFuncs := []*ToolFunc{}
		// 	// fn:agent__agent_command
		// 	for _, v := range r.AgentToolMap {
		// 		// skip agent itself
		// 		if name == v.Service {
		// 			continue
		// 		}
		// 		agentFuncs = append(agentFuncs, v)
		// 	}
		// 	return agentFuncs, nil
		// }
		// for _, v := range r.AgentToolMap {
		// 	if v.Service == s {
		// 		return []*ToolFunc{v}, nil
		// 	}
		// }
		// return nil, fmt.Errorf("no such agent tool: %s", s)
	}

	getTool := func(s string) (*ToolFunc, error) {
		for _, v := range r.Vars.ToolMap {
			if v.Func == s {
				return v, nil
			}
		}
		return nil, fmt.Errorf("no such tool: %s", s)
	}

	getAgentConfig := func(s string) (*AgentConfig, error) {
		for _, a := range config.Agents {
			if a.Name == s {
				return &a, nil
			}
		}
		return nil, fmt.Errorf("no such agent: %s", s)
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
			// mcp:server
			if strings.HasPrefix(f, "mcp:") {
				mcpFuncs, err := getMcpTools(f)
				if err != nil {
					return nil, err
				}
				for _, fn := range mcpFuncs {
					funcMap[fn.Name()] = fn
				}
				continue
			}

			// agent tools
			if strings.HasPrefix(f, "agent:") {
				agentFuncs, err := getAgentTools(f)
				if err != nil {
					return nil, err
				}
				for _, fn := range agentFuncs {
					funcMap[fn.Name()] = fn
				}
				continue
			}

			// system tools
			if strings.HasPrefix(f, "system:") {
				sysFuncs, err := getSystemTools()
				if err != nil {
					return nil, err
				}
				for _, fn := range sysFuncs {
					funcMap[fn.Name()] = fn
				}
				continue
			}

			// builtin functions
			fn, err := getTool(f)
			if err != nil {
				return nil, err
			}
			funcMap[fn.Name()] = fn
		}

		var funcs []*ToolFunc
		for _, v := range funcMap {
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

	creator := func(name string, vars *Vars) (*Agent, error) {
		agentCfg, err := getAgentConfig(name)
		if err != nil {
			return nil, err
		}
		var deps []*Agent

		if len(agentCfg.Dependencies) > 0 {
			for _, dep := range agentCfg.Dependencies {
				depCfg, err := getAgentConfig(dep)
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

	return creator(name, r.Vars)
}
