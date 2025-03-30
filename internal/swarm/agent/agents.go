package agent

import (
	_ "embed"
	"strings"

	"github.com/qiangli/ai/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
	"github.com/qiangli/ai/internal/swarm/agent/resource"
)

var agentConfigMap = map[string][][]byte{}
var agentToolMap = map[string]*api.ToolFunc{}

func initAgents(app *api.AppConfig) {
	resourceMap := resource.AgentCommandMap
	for k, v := range resourceMap {
		agentConfigMap[k] = [][]byte{resource.CommonData, v.Data}
	}

	// skip internal as tool - e.g launch
	for _, v := range resourceMap {
		if !app.Internal && v.Internal {
			continue
		}
		parts := strings.SplitN(v.Name, "/", 2)
		var service = parts[0]
		var toolName string
		if len(parts) == 2 {
			toolName = parts[1]
		}

		fn := &api.ToolFunc{
			Type:        swarm.ToolTypeAgent,
			Kit:         service,
			Name:        toolName,
			Description: v.Description,
		}
		agentToolMap[fn.ID()] = fn
	}
}

var resourceMap = resource.Prompts

// init agents/tools
func InitApp(app *api.AppConfig) {
	initAgents(app)
	swarm.InitTools(app)
}

func RunSwarm(cfg *api.AppConfig, input *api.UserInput) error {
	name := input.Agent
	command := input.Command
	log.Debugf("Running agent %q %s with swarm\n", name, command)

	//
	InitApp(cfg)

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
	toolMap := make(map[string]*api.ToolFunc)
	tools, err := listTools(cfg.McpServerUrl)
	if err != nil {
		return err
	}
	for _, v := range tools {
		toolMap[v.ID()] = v
	}
	sw.Vars.ToolRegistry = toolMap

	showInput(cfg, input)

	req := &api.Request{
		Agent:    name,
		Command:  command,
		RawInput: input,
	}
	resp := &api.Response{}

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

		cfg.Stdout = cfg.Stdout + v.Content
	}

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)

	return nil
}

func showInput(cfg *api.AppConfig, input *api.UserInput) {
	PrintInput(cfg, input)
}

func processOutput(cfg *api.AppConfig, message *api.Output) {
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
		v.Type = "agent"
		tools = append(tools, v)
	}
	return tools, nil
}

// ListTools returns a list of all available tools, including agent, mcp, system, and function tools.
// This is for CLI
func listTools(mcpServerUrl string) ([]*api.ToolFunc, error) {
	list := []*api.ToolFunc{}

	// agent tools
	agentTools, err := ListAgentTools()
	if err != nil {
		return nil, err
	}
	list = append(list, agentTools...)

	// mcp tools
	mcpTools, err := swarm.ListMcpTools(mcpServerUrl)
	if err != nil {
		return nil, err
	}
	for _, v := range mcpTools {
		list = append(list, v...)
	}

	// system and misc tools
	sysTools := swarm.ListTools()
	list = append(list, sysTools...)

	// function tools
	funcTools, err := ListFuncTools()
	if err != nil {
		return nil, err
	}
	list = append(list, funcTools...)

	return list, nil
}

// ListTools returns a list of all available tools for exporting (mcp and system tools).
func ListServiceTools(mcpServerUrl string) ([]*api.ToolFunc, error) {
	list := []*api.ToolFunc{}

	// mcp tools
	mcpTools, err := swarm.ListMcpTools(mcpServerUrl)
	if err != nil {
		return nil, err
	}
	for _, v := range mcpTools {
		list = append(list, v...)
	}

	// system and misc tools
	sysTools := swarm.ListTools()
	list = append(list, sysTools...)

	return list, nil
}
