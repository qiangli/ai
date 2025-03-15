package agent

import (
	_ "embed"
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
	for _, v := range resourceMap {
		if v.Internal {
			continue
		}
		parts := strings.SplitN(v.Name, "/", 2)
		var service = parts[0]
		var toolName string
		if len(parts) == 2 {
			toolName = parts[1]
		}

		fn := &api.ToolFunc{
			Label:       "agent",
			Service:     service,
			Func:        toolName,
			Description: v.Description,
		}
		agentToolMap[fn.Name()] = fn
	}
}

var resourceMap = resource.Prompts

func RunSwarm(cfg *internal.AppConfig, input *api.UserInput) error {
	name := input.Agent
	log.Debugf("Running agent %q with swarm\n", name)

	//
	sw, err := swarm.NewSwarm(cfg)
	if err != nil {
		return err
	}

	sw.AgentConfigMap = agentConfigMap
	sw.ResourceMap = resourceMap
	sw.TemplateFuncMap = tplFuncMap
	sw.AdviceMap = adviceMap
	sw.EntrypointMap = entrypointMap

	//
	sw.Vars.FuncRegistry = funcRegistry
	//
	toolMap := make(map[string]*swarm.ToolFunc)
	tools, err := listTools(cfg.McpServerUrl)
	if err != nil {
		return err
	}
	for _, v := range tools {
		toolMap[v.Name()] = v
	}
	sw.Vars.ToolMap = toolMap

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

func ListAgentTools() ([]*api.ToolFunc, error) {
	tools := make([]*api.ToolFunc, 0)
	for _, v := range agentToolMap {
		v.Label = "agent"
		tools = append(tools, v)
	}
	return tools, nil
}
