package swarm

// import (
// 	"fmt"
// 	"strings"

// 	"github.com/qiangli/ai/swarm/api"
// )

// func CreateAgent(vars *api.Vars, name, command string, input *api.UserInput) (*Agent, error) {
// 	config, err := LoadAgents(vars.Config, name, input)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// config := r.Config
// 	adviceMap := vars.AdviceMap

// 	getTools := func(toolType string) ([]*api.ToolFunc, error) {
// 		var list []*api.ToolFunc
// 		for _, v := range vars.ToolRegistry {
// 			if toolType == "*" || v.Type == toolType {
// 				list = append(list, v)
// 			}
// 		}
// 		return list, nil
// 	}

// 	getTool := func(s string) (*api.ToolFunc, error) {
// 		for _, v := range vars.ToolRegistry {
// 			if v.Name == s {
// 				return v, nil
// 			}
// 		}
// 		return nil, fmt.Errorf("no such tool: %s", s)
// 	}

// 	getAgentConfig := func(n, c string) (*api.AgentConfig, error) {
// 		// check for more specific agent first
// 		if c != "" {
// 			ap := fmt.Sprintf("%s/%s", n, c)
// 			for _, a := range config.Agents {
// 				if a.Name == ap {
// 					return &a, nil
// 				}
// 			}
// 		}
// 		for _, a := range config.Agents {
// 			if a.Name == n {
// 				return &a, nil
// 			}
// 		}
// 		return nil, fmt.Errorf("no such agent: %s / %s", n, c)
// 	}

// 	newAgent := func(ac *api.AgentConfig, vars *api.Vars) (*Agent, error) {
// 		agent := Agent{
// 			Name:        ac.Name,
// 			Display:     ac.Display,
// 			Role:        ac.Instruction.Role,
// 			Instruction: ac.Instruction.Content,
// 			// Vars:        vars,
// 			RawInput: input,
// 			MaxTurns: config.MaxTurns,
// 			MaxTime:  config.MaxTime,
// 		}

// 		// override from command line flags
// 		// if r.AppConfig.Role != "" {
// 		// 	agent.Role = r.AppConfig.Role
// 		// }
// 		// if r.AppConfig.Prompt != "" {
// 		// 	agent.Instruction = r.AppConfig.Prompt
// 		// }
// 		// if r.AppConfig.MaxTurns != 0 {
// 		// 	agent.MaxTurns = r.AppConfig.MaxTurns
// 		// }
// 		// if r.AppConfig.MaxTime != 0 {
// 		// 	agent.MaxTime = r.AppConfig.MaxTime
// 		// }

// 		level := toModelLevel(ac.Model)
// 		model, ok := vars.Models[level]
// 		if !ok {
// 			return nil, fmt.Errorf("no such model: %s", ac.Model)
// 		}
// 		agent.Model = model

// 		// tools
// 		funcMap := make(map[string]*api.ToolFunc)
// 		for _, f := range ac.Functions {
// 			if f == "*" {
// 				funcs, err := getTools("*")
// 				if err != nil {
// 					return nil, err
// 				}
// 				for _, fn := range funcs {
// 					funcMap[fn.ID()] = fn
// 				}
// 				continue
// 			}
// 			// type:*
// 			if strings.Contains(f, ":") {
// 				parts := strings.SplitN(f, ":", 2)
// 				if len(parts) > 0 {
// 					funcs, err := getTools(parts[0])
// 					if err != nil {
// 						return nil, err
// 					}
// 					for _, fn := range funcs {
// 						funcMap[fn.ID()] = fn
// 					}
// 					continue
// 				}
// 			}

// 			// builtin functions
// 			fn, err := getTool(f)
// 			if err != nil {
// 				return nil, err
// 			}
// 			funcMap[fn.ID()] = fn
// 		}

// 		// FIXME
// 		// 1. better handle this to avoid agent calling self as tools
// 		// filter out all agent tools from the agent itself
// 		// 2. handle namespace to avoid collision of tool names
// 		agentName := ac.Name
// 		if strings.Contains(agentName, "/") {
// 			parts := strings.SplitN(agentName, "/", 2)
// 			agentName = parts[0]
// 		}
// 		var funcs []*api.ToolFunc
// 		for _, v := range funcMap {
// 			if v.Type == ToolTypeAgent && v.Kit == agentName {
// 				continue
// 			}
// 			funcs = append(funcs, v)
// 		}
// 		agent.Tools = funcs

// 		if ac.Advices.Before != "" {
// 			if ad, ok := adviceMap[ac.Advices.Before]; ok {
// 				agent.BeforeAdvice = ad
// 			} else {
// 				return nil, fmt.Errorf("no such advice: %s", ac.Advices.Before)
// 			}
// 		}
// 		if ac.Advices.After != "" {
// 			if ad, ok := adviceMap[ac.Advices.After]; ok {
// 				agent.AfterAdvice = ad
// 			} else {
// 				return nil, fmt.Errorf("no such advice: %s", ac.Advices.After)
// 			}
// 		}
// 		if ac.Advices.Around != "" {
// 			if ad, ok := adviceMap[ac.Advices.Around]; ok {
// 				agent.AroundAdvice = ad
// 			} else {
// 				return nil, fmt.Errorf("no such advice: %s", ac.Advices.Around)
// 			}
// 		}
// 		if ac.Entrypoint != "" {
// 			if ep, ok := vars.EntrypointMap[ac.Entrypoint]; ok {
// 				agent.Entrypoint = ep
// 			} else {
// 				return nil, fmt.Errorf("no such entrypoint: %s", ac.Entrypoint)
// 			}
// 		}

// 		return &agent, nil
// 	}

// 	creator := func(vars *api.Vars, name, command string) (*Agent, error) {
// 		agentCfg, err := getAgentConfig(name, command)
// 		if err != nil {
// 			return nil, err
// 		}
// 		var deps []*Agent

// 		if len(agentCfg.Dependencies) > 0 {
// 			for _, dep := range agentCfg.Dependencies {
// 				depCfg, err := getAgentConfig(dep, "")
// 				if err != nil {
// 					return nil, err
// 				}
// 				agent, err := newAgent(depCfg, vars)
// 				if err != nil {
// 					return nil, err
// 				}
// 				deps = append(deps, agent)
// 			}
// 		}
// 		agent, err := newAgent(agentCfg, vars)
// 		if err != nil {
// 			return nil, err
// 		}
// 		agent.Dependencies = deps

// 		return agent, nil
// 	}

// 	return creator(vars, name, command)
// }

// func LoadAgents(app *api.AppConfig, name string, input *api.UserInput) (*api.AgentsConfig, error) {
// 	// if config != nil && len(config.Agents) > 0 {
// 	// 	for _, a := range config.Agents {
// 	// 		if a.Name == name {
// 	// 			return nil
// 	// 		}
// 	// 	}
// 	// }

// 	// data, ok := r.AgentConfigMap[name]
// 	// if !ok {
// 	// 	return internal.NewUserInputError("not supported yet: " + name)
// 	// }
// 	// err := loadAgentsData(data)
// 	// if err != nil {
// 	// 	return err
// 	// }

// 	// override
// 	// app := r.AppConfig
// 	// config := r.Config

// 	// var modelMap = make(map[api.Level]*api.Model)
// 	// for _, m := range config.Models {
// 	// 	if m.External {
// 	// 		switch m.Name {
// 	// 		case "L1":
// 	// 			modelMap[api.L1] = api.Level1(app.LLM)
// 	// 		case "L2":
// 	// 			modelMap[api.L2] = api.Level2(app.LLM)
// 	// 		case "L3":
// 	// 			modelMap[api.L3] = api.Level3(app.LLM)
// 	// 		case "Image":
// 	// 			modelMap[api.LImage] = api.ImageModel(app.LLM)
// 	// 		}
// 	// 	} else {
// 	// 		l := toModelLevel(m.Name)
// 	// 		modelMap[l] = &api.Model{
// 	// 			Type:    api.ModelType(m.Type),
// 	// 			Name:    m.Model,
// 	// 			BaseUrl: m.BaseUrl,
// 	// 			ApiKey:  m.ApiKey,
// 	// 		}
// 	// 	}
// 	// }
// 	// r.Vars.Models = modelMap

// 	return nil, nil
// }
