package agent

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
	"github.com/qiangli/ai/internal/util"
)

const LaunchAgent = "launch"

var resourceMap = resource.Prompts

//go:embed resource/agents.yaml
var agentsYaml []byte

type UserConfig struct {
	Name    string `yaml:"name"`
	Display string `yaml:"display"`
}

type AgentsConfig struct {
	User      UserConfig       `yaml:"user"`
	Agents    []AgentConfig    `yaml:"agents"`
	Functions []FunctionConfig `yaml:"functions"`
	Models    []ModelConfig    `yaml:"models"`
}

type AgentConfig struct {
	Name        string `yaml:"name"`
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	Role        string `yaml:"role"`
	Instruction string `yaml:"instruction"`
	Model       string `yaml:"model"`

	Entrypoint string `yaml:"entrypoint"`

	Functions []string `yaml:"functions"`

	Dependencies []string `yaml:"dependencies"`

	Advices AdviceConfig `yaml:"advices"`
}

type FunctionConfig struct {
	Type   string   `yaml:"type"`
	Labels []string `yaml:"labels"`

	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`
}

type ModelConfig struct {
	Name        string `yaml:"name"`
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

// Load the agents configuration from the embedded yaml file
func LoadAgentsConfig() (*AgentsConfig, error) {
	var cfg AgentsConfig

	err := yaml.Unmarshal(agentsYaml, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Run(app *internal.AppConfig, starter string, input *UserInput) error {
	log.Debugf("Running agent %q with swarm\n", starter)

	//
	config, err := LoadAgentsConfig()
	if err != nil {
		return err
	}

	var modelMap = make(map[string]*api.Model)
	for _, m := range config.Models {
		if m.External {
			switch m.Name {
			case "L1":
				modelMap["L1"] = internal.Level1(app.LLM)
			case "L2":
				modelMap["L2"] = internal.Level2(app.LLM)
			case "L3":
				modelMap["L3"] = internal.Level3(app.LLM)
			case "Image":
				modelMap["Image"] = internal.ImageModel(app.LLM)
			}
		} else {
			modelMap[m.Name] = &api.Model{
				Name:    m.Model,
				BaseUrl: m.BaseUrl,
				ApiKey:  m.ApiKey,
			}
		}
	}

	var functionMap = make(map[string]FunctionConfig)
	for _, f := range config.Functions {
		functionMap[f.Name] = f
	}

	// "resource:" prefix is used to refer to a resource
	// "vars:" prefix is used to refer to a variable
	apply := func(s string, vars *swarm.Vars) (string, error) {
		if strings.HasPrefix(s, "resource:") {
			v, ok := resourceMap[s[9:]]
			if !ok {
				return "", fmt.Errorf("no such resource: %s", s[9:])
			}
			return applyTemplate(v, vars)
		}
		if strings.HasPrefix(s, "vars:") {
			v := vars.GetString(s[5:])
			return v, nil
		}
		return s, nil
	}

	getAgent := func(name string) (*AgentConfig, error) {
		for _, a := range config.Agents {
			if a.Name == name {
				return &a, nil
			}
		}
		return nil, fmt.Errorf("no such agent: %s", name)
	}

	newAgent := func(ac *AgentConfig, vars *swarm.Vars) (*swarm.Agent, error) {
		model, ok := modelMap[ac.Model]
		if !ok {
			return nil, fmt.Errorf("no such model: %s", ac.Model)
		}
		content, err := apply(ac.Instruction, vars)
		if err != nil {
			return nil, err
		}
		functions := []*swarm.ToolFunc{}
		for _, f := range ac.Functions {
			fn, ok := functionMap[f]
			if !ok {
				return nil, fmt.Errorf("no such function: %s", f)
			}
			functions = append(functions, &swarm.ToolFunc{
				Name:        fn.Name,
				Description: fn.Description,
				Parameters:  fn.Parameters,
			})
		}
		agent := swarm.Agent{
			Name:    ac.Name,
			Display: ac.Display,
			Model: &swarm.Model{
				Name:    model.Name,
				BaseUrl: model.BaseUrl,
				ApiKey:  model.ApiKey,
			},
			Role:        ac.Role,
			Instruction: content,
			Functions:   functions,
		}
		if ac.Advices.Before != "" {
			if ad, ok := adviceMap[ac.Advices.Before]; ok {
				agent.BeforeAdvice = ad
			}
		}
		if ac.Advices.After != "" {
			if ad, ok := adviceMap[ac.Advices.After]; ok {
				agent.AfterAdvice = ad
			}
		}
		if ac.Advices.Around != "" {
			if ad, ok := adviceMap[ac.Advices.Around]; ok {
				agent.AroundAdvice = ad
			}
		}

		//
		if log.IsVerbose() {
			// add logging around advice
			log.Debugf("Agent %+v\n", agent)
		}
		if internal.DryRun {
			agent.AroundAdvice = func(_ *swarm.Request, resp *swarm.Response, next swarm.Advice) error {
				log.Debugf("Before agent %s\n", agent.Name)
				resp.Messages = []*swarm.Message{
					{
						Content: internal.DryRunContent,
						Sender:  "dry-run",
					},
				}
				resp.Agent = &agent
				resp.Vars = vars
				log.Debugf("After agent %s\n", agent.Name)
				return nil
			}
		}
		return &agent, nil
	}

	// initialize
	agentCfg, err := getAgent(starter)
	if err != nil {
		return err
	}

	vars := swarm.NewVars()
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return err
	}
	vars.Arch = sysInfo.Arch
	vars.OS = sysInfo.OS
	vars.ShellInfo = sysInfo.ShellInfo
	vars.OSInfo = sysInfo.OSInfo
	vars.UserInfo = sysInfo.UserInfo
	vars.WorkDir = sysInfo.WorkDir

	// TODO refactor Input in Vars or Request
	vars.Agent = input.Agent
	vars.Subcommand = input.Subcommand
	vars.Input = input.Input()
	vars.Intent = input.Intent()
	vars.Query = input.Query()
	vars.Files = input.Files

	var history = []*swarm.Message{}
	msg := &swarm.Message{Role: swarm.RoleUser, Content: input.Input(), Sender: config.User.Name}
	req := &swarm.Request{Message: msg, Vars: vars}

	if len(agentCfg.Dependencies) > 0 {
		for _, dep := range agentCfg.Dependencies {
			depCfg, err := getAgent(dep)
			if err != nil {
				return err
			}
			agent, err := newAgent(depCfg, vars)
			if err != nil {
				return err
			}

			s := swarm.NewSwarm(agent, vars, history)
			resp, err := s.Run(req)
			if err != nil {
				return err
			}
			history = resp.Messages

			log.Debugf("%+v\n", resp)
		}
	}

	agent, err := newAgent(agentCfg, vars)
	if err != nil {
		return err
	}

	hist := []*swarm.Message{}

	s := swarm.NewSwarm(agent, vars, hist)

	resp, err := s.Run(req)
	if err != nil {
		return err
	}

	log.Debugf("Agent %+v\n", resp.Agent)
	for _, m := range resp.Messages {
		log.Debugf("Message %+v\n", m)
	}

	processContent(app, &api.Response{
		Agent:       resp.Agent.Display,
		ContentType: api.ContentTypeText,
		Content:     resp.LastMessage().Content,
	})

	log.Debugf("Agent task completed: %s %v\n", app.Command, app.Args)

	return nil
}
