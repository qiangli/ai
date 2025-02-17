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

const launchAgent = "launch"

var agentsRunMap = map[string]func(*internal.AppConfig, string, *UserInput) error{
	"ask":    RunAskAgent,
	"script": RunScriptAgent,
	"git":    RunGitAgent,
	"pr":     RunPrAgent,
	"gptr":   RunGptrAgent,
	"seek":   RunGptrAgent,
	"aider":  RunAiderAgent,
	"oh":     RunOhAgent,
	"sql":    RunSqlAgent,
	"doc":    RunDocAgent,
	"eval":   RunEvalAgent,
	"draw":   RunDrawAgent,
}

var resourceMap = resource.Prompts

//go:embed resource/common.yaml
var configCommonYaml []byte

//go:embed resource/launch/agent.yaml
var configLaunchAgentYaml []byte

func RunLaunchAgent(app *internal.AppConfig, input *UserInput) error {
	data := [][]byte{configLaunchAgentYaml}
	return runSwarm(app, data, launchAgent, input)
}

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

//go:embed resource/sql/agent.yaml
var configSqlAgentYaml []byte

func RunSqlAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configSqlAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/doc/agent.yaml
var configDocAgentYaml []byte

func RunDocAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configDocAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/eval/agent.yaml
var configEvalAgentYaml []byte

func RunEvalAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configCommonYaml, configEvalAgentYaml}
	return runSwarm(app, data, agent, input)
}

//go:embed resource/draw/agent.yaml
var configDrawAgentYaml []byte

func RunDrawAgent(app *internal.AppConfig, name string, input *UserInput) error {
	var agent = baseCommand(input.Subcommand)
	if agent == "" {
		agent = name
	}
	data := [][]byte{configDrawAgentYaml}
	return runSwarm(app, data, agent, input)
}

// LoadAgentsConfig loads the agent configuration from the provided YAML data.
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
				Type:    api.ModelType(m.Type),
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

	//
	sw.Vars.Input = input
	sw.Vars.Role = app.Role
	sw.Vars.Prompt = app.Prompt

	if app.Db != nil {
		sw.Vars.DBCred = &swarm.DBCred{
			Host:     app.Db.Host,
			Port:     app.Db.Port,
			Username: app.Db.Username,
			Password: app.Db.Password,
			DBName:   app.Db.DBName,
		}
	}
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
		m := &api.Response{
			Agent:       name,
			ContentType: v.ContentType,
			Content:     v.Content,
		}
		processContent(app, m)
	}

	log.Debugf("Agent task completed: %s %v\n", app.Command, app.Args)

	return nil
}
