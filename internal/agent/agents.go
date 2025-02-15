package agent

import (
	_ "embed"

	"dario.cat/mergo"
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

//go:embed resource/common.yaml
var configCommonYaml []byte

//go:embed resource/ask/agent.yaml
var configAskAgentYaml []byte

func RunAskAgent(app *internal.AppConfig, agent string, input *UserInput) error {
	data := [][]byte{configAskAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/script/agent.yaml
var configScriptAgentYaml []byte

func RunScriptAgent(app *internal.AppConfig, agent string, input *UserInput) error {
	data := [][]byte{configScriptAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/git/agent.yaml
var configGitAgentYaml []byte

func RunGitAgent(app *internal.AppConfig, name string, input *UserInput) error {
	// agent name is same as subcommand
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configGitAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/pr/agent.yaml
var configPrAgentYaml []byte

func RunPrAgent(app *internal.AppConfig, name string, input *UserInput) error {
	// agent name is same as subcommand
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configPrAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/gptr/agent.yaml
var configGptrAgentYaml []byte

func RunGptrAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configGptrAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/oh/agent.yaml
var configOhAgentYaml []byte

func RunOhAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configCommonYaml, configOhAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/aider/agent.yaml
var configAiderAgentYaml []byte

func RunAiderAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configCommonYaml, configAiderAgentYaml}
	return runSwarm(app, data, agent, input)
}

func LoadAgentsConfig(data [][]byte) (*swarm.AgentsConfig, error) {
	merged := &swarm.AgentsConfig{}

	for _, v := range data {
		cfg := &swarm.AgentsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	merged.ResourceMap = resourceMap
	merged.TemplateFuncMap = tplFuncMap

	// TODO per agent?
	merged.AdviceMap = adviceMap
	merged.EntrypointMap = entrypointMap

	return merged, nil
}

func runSwarm(app *internal.AppConfig, data [][]byte, starter string, input *UserInput) error {
	log.Debugf("Running agent %q with swarm\n", starter)

	//
	config, err := LoadAgentsConfig(data)
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

	var functionMap = make(map[string]*swarm.ToolFunc)
	for _, v := range config.Functions {
		functionMap[v.Name] = &swarm.ToolFunc{
			Name:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
		}
	}

	//
	sw := swarm.NewSwarm(config)
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return err
	}

	sw.Vars.Input = input

	//
	sw.Vars.Arch = sysInfo.Arch
	sw.Vars.OS = sysInfo.OS
	sw.Vars.ShellInfo = sysInfo.ShellInfo
	sw.Vars.OSInfo = sysInfo.OSInfo
	sw.Vars.UserInfo = sysInfo.UserInfo
	sw.Vars.WorkDir = sysInfo.WorkDir
	//
	sw.Vars.Models = modelMap
	sw.Vars.Functions = functionMap
	sw.Vars.FuncRegistry = funcRegistry

	resp := &swarm.Response{}
	if err := sw.Run(&swarm.Request{
		Agent:    starter,
		RawInput: input,
	}, resp); err != nil {
		return err
	}

	log.Debugf("Agent %+v\n", resp.Agent)
	for _, m := range resp.Messages {
		log.Debugf("Message %+v\n", m)
	}
	var name = ""
	if resp.Agent != nil {
		name = resp.Agent.Display
	}

	results := resp.Messages
	for _, v := range results {
		processContent(app, &api.Response{
			Agent:       name,
			ContentType: api.ContentTypeText,
			Content:     v.Content,
		})
	}

	log.Debugf("Agent task completed: %s %v\n", app.Command, app.Args)

	return nil
}
