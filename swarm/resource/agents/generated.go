//DO NOT EDIT. This file is generated.
package resource

import _ "embed"

//go:embed prompts/agent_meta_system_role.md
var agent_meta_system_role string

//go:embed prompts/docker_input_user_role.md
var docker_input_user_role string


var Prompts = map[string]string{
	"agent_meta_system_role": agent_meta_system_role,
	"docker_input_user_role": docker_input_user_role,
}

//go:embed common.yaml
var CommonData []byte

type AgentConfig struct {
	Name        string
	Description string
	Overview    string
	Internal    bool
	Data		[]byte
}



//go:embed agent/agent.yaml
var agent_agent_yaml_data []byte

//go:embed aider/agent.yaml
var aider_agent_yaml_data []byte

//go:embed ask/agent.yaml
var ask_agent_yaml_data []byte

//go:embed chdir/agent.yaml
var chdir_agent_yaml_data []byte

//go:embed doc/agent.yaml
var doc_agent_yaml_data []byte

//go:embed draw/agent.yaml
var draw_agent_yaml_data []byte

//go:embed eval/agent.yaml
var eval_agent_yaml_data []byte

//go:embed exec/agent.yaml
var exec_agent_yaml_data []byte

//go:embed find/agent.yaml
var find_agent_yaml_data []byte

//go:embed git/agent.yaml
var git_agent_yaml_data []byte

//go:embed github/agent.yaml
var github_agent_yaml_data []byte

//go:embed gptr/agent.yaml
var gptr_agent_yaml_data []byte

//go:embed mcp/agent.yaml
var mcp_agent_yaml_data []byte

//go:embed oh/agent.yaml
var oh_agent_yaml_data []byte

//go:embed pr/agent.yaml
var pr_agent_yaml_data []byte

//go:embed shell/agent.yaml
var shell_agent_yaml_data []byte

//go:embed sql/agent.yaml
var sql_agent_yaml_data []byte

//go:embed web/agent.yaml
var web_agent_yaml_data []byte

//go:embed workspace/agent.yaml
var workspace_agent_yaml_data []byte

var AgentCommandMap = map[string]AgentConfig {
	"agent": {
Name: "agent",
Description: "Dispatch to the most appropriate agent based on the user's input.",
Internal: false,
Data: agent_agent_yaml_data,
Overview: "",
},
	"aider": {
Name: "aider",
Description: "Integrate LLMs for collaborative coding, refactoring, bug fixing, and test development.",
Internal: false,
Data: aider_agent_yaml_data,
Overview: "",
},
	"ask": {
Name: "ask",
Description: "Deliver concise, reliable answers on a wide range of topics.",
Internal: false,
Data: ask_agent_yaml_data,
Overview: "",
},
	"chdir": {
Name: "chdir",
Description: "Evaluate users input and locate the directory on the local system the user intends to change to.",
Internal: false,
Data: chdir_agent_yaml_data,
Overview: "",
},
	"doc": {
Name: "doc",
Description: "Create a polished document by integrating draft materials into the provided template.",
Internal: false,
Data: doc_agent_yaml_data,
Overview: "",
},
	"doc/archive": {
Name: "doc/archive",
Description: "Compress or decompress files using various tools",
Internal: false,
Data: doc_agent_yaml_data,
Overview: "",
},
	"draw": {
Name: "draw",
Description: "Generate images based on user input, providing visual representations of text-based descriptions.",
Internal: false,
Data: draw_agent_yaml_data,
Overview: "",
},
	"eval": {
Name: "eval",
Description: "Evaluate and test tools.",
Internal: false,
Data: eval_agent_yaml_data,
Overview: "",
},
	"exec": {
Name: "exec",
Description: "Execute commands based on user instructions",
Internal: false,
Data: exec_agent_yaml_data,
Overview: "",
},
	"find": {
Name: "find",
Description: "Dynamically select tools for efficient local file and text searches based on user queries",
Internal: false,
Data: find_agent_yaml_data,
Overview: "",
},
	"git": {
Name: "git",
Description: "Generate git commit message based on users input and the provided diffs.",
Internal: true,
Data: git_agent_yaml_data,
Overview: "",
},
	"git/long": {
Name: "git/long",
Description: "Generate git commit messages based on the provided diffs using the Conventional Commits specification",
Internal: false,
Data: git_agent_yaml_data,
Overview: "",
},
	"git/short": {
Name: "git/short",
Description: "Generate concise, one-line git commit messages based on the provided diffs.",
Internal: false,
Data: git_agent_yaml_data,
Overview: "",
},
	"github": {
Name: "github",
Description: "help user manage alerts, repository content, issues, and pull requests",
Internal: false,
Data: github_agent_yaml_data,
Overview: "",
},
	"gptr": {
Name: "gptr",
Description: "Deliver live, realtime, accurate, relevant insights from diverse online sources.",
Internal: false,
Data: gptr_agent_yaml_data,
Overview: "",
},
	"mcp": {
Name: "mcp",
Description: "Simple MCP agent",
Internal: false,
Data: mcp_agent_yaml_data,
Overview: "",
},
	"oh": {
Name: "oh",
Description: "Engineering assistant promoting incremental development and detailed refactoring support.",
Internal: false,
Data: oh_agent_yaml_data,
Overview: "",
},
	"pr": {
Name: "pr",
Description: "Enhance PR management with automated summaries, reviews, suggestions, and changelog updates.",
Internal: false,
Data: pr_agent_yaml_data,
Overview: "",
},
	"pr/changelog": {
Name: "pr/changelog",
Description: "Update the CHANGELOG.md file with the PR changes",
Internal: false,
Data: pr_agent_yaml_data,
Overview: "",
},
	"pr/describe": {
Name: "pr/describe",
Description: "Generate PR description - title, type, summary, code walkthrough and labels",
Internal: false,
Data: pr_agent_yaml_data,
Overview: "",
},
	"pr/improve": {
Name: "pr/improve",
Description: "Provide code suggestions for improving the PR",
Internal: false,
Data: pr_agent_yaml_data,
Overview: "",
},
	"pr/review": {
Name: "pr/review",
Description: "Give feedback about the PR, possible issues, security concerns, review effort and more",
Internal: false,
Data: pr_agent_yaml_data,
Overview: "",
},
	"shell": {
Name: "shell",
Description: "Assist with scripting, command execution, and troubleshooting shell tasks.",
Internal: false,
Data: shell_agent_yaml_data,
Overview: "",
},
	"sql": {
Name: "sql",
Description: "Streamline SQL query generation, helping users derive insights without SQL expertise.",
Internal: false,
Data: sql_agent_yaml_data,
Overview: "",
},
	"web": {
Name: "web",
Description: "Search the web and fetch the content from a URL.",
Internal: false,
Data: web_agent_yaml_data,
Overview: "",
},
	"workspace": {
Name: "workspace",
Description: "Determines the user's workspace based on user's input.",
Internal: false,
Data: workspace_agent_yaml_data,
Overview: "",
},
}
