package agent

import (
	_ "embed"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

const launchAgent = "launch"

var agentConfigMap = map[string][][]byte{
	"launch": [][]byte{configLaunchAgentYaml},
	"ask":    [][]byte{configAskAgentYaml},
	"script": [][]byte{configScriptAgentYaml},
	"git":    [][]byte{configGitAgentYaml},
	"pr":     [][]byte{configPrAgentYaml},
	"gptr":   [][]byte{configGptrAgentYaml},
	"seek":   [][]byte{configGptrAgentYaml},
	"aider":  [][]byte{configCommonYaml, configAiderAgentYaml},
	"oh":     [][]byte{configCommonYaml, configOhAgentYaml},
	"sql":    [][]byte{configSqlAgentYaml},
	"doc":    [][]byte{configDocAgentYaml},
	"eval":   [][]byte{configCommonYaml, configEvalAgentYaml},
	"draw":   [][]byte{configDrawAgentYaml},
}

var resourceMap = resource.Prompts

//go:embed resource/common.yaml
var configCommonYaml []byte

//go:embed resource/launch/agent.yaml
var configLaunchAgentYaml []byte

//go:embed resource/ask/agent.yaml
var configAskAgentYaml []byte

//go:embed resource/script/agent.yaml
var configScriptAgentYaml []byte

//go:embed resource/git/agent.yaml
var configGitAgentYaml []byte

//go:embed resource/pr/agent.yaml
var configPrAgentYaml []byte

//go:embed resource/gptr/agent.yaml
var configGptrAgentYaml []byte

//go:embed resource/oh/agent.yaml
var configOhAgentYaml []byte

//go:embed resource/aider/agent.yaml
var configAiderAgentYaml []byte

//go:embed resource/sql/agent.yaml
var configSqlAgentYaml []byte

//go:embed resource/doc/agent.yaml
var configDocAgentYaml []byte

//go:embed resource/eval/agent.yaml
var configEvalAgentYaml []byte

//go:embed resource/draw/agent.yaml
var configDrawAgentYaml []byte

func runSwarm(app *internal.AppConfig, name string, input *UserInput) error {
	log.Debugf("Running agent %q with swarm\n", name)

	sw, err := swarm.NewSwarm(app)
	if err != nil {
		return err
	}

	sw.AgentConfigMap = agentConfigMap
	sw.FuncRegistry = funcRegistry
	sw.ResourceMap = resourceMap
	sw.TemplateFuncMap = tplFuncMap
	sw.AdviceMap = adviceMap
	sw.EntrypointMap = entrypointMap

	sw.Agent = name

	// if err := sw.Load(name, input); err != nil {
	// 	return err
	// }

	req := &swarm.Request{
		Agent:    name,
		RawInput: input,
	}
	resp := &swarm.Response{}

	if err := sw.Run(req, resp); err != nil {
		return err
	}

	log.Debugf("Agent %+v\n", resp.Agent)
	for _, m := range resp.Messages {
		log.Debugf("Message %+v\n", m)
	}

	var display = ""
	if resp.Agent != nil {
		display = resp.Agent.Display
	}

	results := resp.Messages
	for _, v := range results {
		m := &api.Response{
			Agent:       display,
			ContentType: v.ContentType,
			Content:     v.Content,
		}
		processContent(app, m)
	}

	log.Debugf("Agent task completed: %s %v\n", app.Command, app.Args)

	return nil
}
