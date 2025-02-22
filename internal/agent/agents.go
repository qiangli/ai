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

var agentConfigMap = map[string][][]byte{}

func init() {
	resourceMap := resource.AgentCommandMap
	for k, v := range resourceMap {
		agentConfigMap[k] = [][]byte{resource.CommonData, v.Data}
	}
}

var resourceMap = resource.Prompts

func RunSwarm(app *internal.AppConfig, input *api.UserInput) error {
	name := input.Agent
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

	req := &swarm.Request{
		Agent:    name,
		RawInput: input,
	}
	resp := &swarm.Response{}

	showInput(app, req)

	if err := sw.Run(req, resp); err != nil {
		return err
	}

	log.Debugf("Agent %+v\n", resp.Agent)
	for _, m := range resp.Messages {
		log.Debugf("Message %+v\n", m)
	}

	var agent = name
	var display = ""
	if resp.Agent != nil {
		agent = resp.Agent.Name
		display = resp.Agent.Display
	}

	results := resp.Messages
	for _, v := range results {
		m := &api.Response{
			Agent:       agent,
			Display:     display,
			ContentType: v.ContentType,
			Content:     v.Content,
		}
		processContent(app, m)
	}

	log.Debugf("Agent task completed: %s %v\n", app.Command, app.Args)

	return nil
}
