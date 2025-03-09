package agent

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

var agentConfigMap = map[string][][]byte{}
var agentToolMap = map[string]*swarm.ToolFunc{}

func init() {
	resourceMap := resource.AgentCommandMap
	for k, v := range resourceMap {
		agentConfigMap[k] = [][]byte{resource.CommonData, v.Data}
	}

	// skip internal as tool - e.g launch
	for k, v := range resourceMap {
		if v.Internal {
			continue
		}
		fn := fmt.Sprintf("agent__%s", strings.ReplaceAll(k, "/", "_"))
		agentToolMap[fn] = &api.ToolFunc{
			Name:        v.Name,
			Description: v.Description,
		}
	}
}

var resourceMap = resource.Prompts

func RunSwarm(cfg *internal.AppConfig, input *api.UserInput) error {
	name := input.Agent
	log.Debugf("Running agent %q with swarm\n", name)

	// TODO:
	roots, err := listRoots()
	if err != nil {
		return err
	}
	cfg.Roots = roots

	//
	sw, err := swarm.NewSwarm(cfg)
	if err != nil {
		return err
	}

	sw.AgentConfigMap = agentConfigMap
	sw.AgentToolMap = agentToolMap
	sw.ResourceMap = resourceMap
	sw.TemplateFuncMap = tplFuncMap
	sw.AdviceMap = adviceMap
	sw.EntrypointMap = entrypointMap
	sw.FuncRegistry = funcRegistry

	showInput(cfg, input)

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

	var display = name
	if resp.Agent != nil {
		display = resp.Agent.Display
	}

	results := resp.Messages
	for _, v := range results {
		out := &api.Output{
			Display:     display,
			ContentType: v.ContentType,
			Content:     v.Content,
		}
		processOutput(cfg, out)
	}

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)

	return nil
}

func showInput(cfg *internal.AppConfig, input *api.UserInput) {
	PrintInput(cfg, input)
}

func processOutput(cfg *internal.AppConfig, message *api.Output) {
	if message.ContentType == api.ContentTypeText || message.ContentType == "" {
		processTextContent(cfg, message)
	} else if message.ContentType == api.ContentTypeB64JSON {
		processImageContent(cfg, message)
	} else {
		log.Debugf("Unsupported content type: %s\n", message.ContentType)
	}
}
