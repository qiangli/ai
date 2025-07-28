// DO NOT EDIT. This file is generated.
package resource

import _ "embed"

var Prompts = map[string]string{}

//go:embed common.yaml
var CommonData []byte

type AgentConfig struct {
	Name        string
	Description string
	Overview    string
	Internal    bool
	Data        []byte
}

//go:embed agent/agent.yaml
var agent_agent_yaml_data []byte

//go:embed ask/agent.yaml
var ask_agent_yaml_data []byte

//go:embed shell/agent.yaml
var shell_agent_yaml_data []byte

//go:embed swe/agent.yaml
var swe_agent_yaml_data []byte

//go:embed web/agent.yaml
var web_agent_yaml_data []byte

var AgentCommandMap = map[string]AgentConfig{
	"agent": {
		Name:        "agent",
		Description: "Dispatch to the most appropriate agent based on the user's input.",
		Internal:    false,
		Data:        agent_agent_yaml_data,
		Overview:    "",
	},
	"ask": {
		Name:        "ask",
		Description: "Deliver concise, reliable answers on a wide range of topics.",
		Internal:    false,
		Data:        ask_agent_yaml_data,
		Overview:    "",
	},
	"shell": {
		Name:        "shell",
		Description: "Assist with scripting, command execution, and troubleshooting shell tasks.",
		Internal:    false,
		Data:        shell_agent_yaml_data,
		Overview:    "",
	},
	"swe": {
		Name:        "swe",
		Description: "Act as an expert software developer",
		Internal:    false,
		Data:        swe_agent_yaml_data,
		Overview:    "",
	},
	"web": {
		Name:        "web",
		Description: "Search the web and fetch the content from a URL.",
		Internal:    false,
		Data:        web_agent_yaml_data,
		Overview:    "",
	},
}
