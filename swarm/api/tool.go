package api

import (
// "context"
// "fmt"
)

// Tool kit configuration
type ToolsConfig struct {
	Owner string `yaml:"owner"`

	// kit name
	// Namespace:
	//
	// func class
	// Agent name
	// MCP server name
	// Virtual file system name
	// Container name
	// Virtual machine name
	// Tool/function (Gemini)
	Kit string `yaml:"kit"`

	// func (server) | system (client) | remote
	Type string `yaml:"type"`

	Connector *ConnectorConfig `yaml:"connector"`

	// system commands used by tools
	Commands []string `yaml:"commands"`

	Tools []*ToolConfig `yaml:"tools"`
}

type ToolConfig struct {
	Kit string `yaml:"kit"`

	Type string `yaml:"type"`

	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`

	Body string `yaml:"body"`

	Condition *ToolCondition `yaml:"condition"`
}

// TODO condidtion needs to be met for tools to be enabled
type ToolCondition struct {
	// required env list
	Env []string `yaml:"env"`

	// found on PATH
	Lookup *string `yaml:"lookup"`

	// shell required
	Shell map[string]any `yaml:"shell"`
}

// ToolDescriptor is a description of a tool function.
type ToolDescriptor struct {
	Name        string
	Description string
	Parameters  map[string]any

	Body string
}

// type ToolFunc struct {
// 	// Namespace:
// 	//
// 	// func class
// 	// Agent name
// 	// MCP server name
// 	// Virtual file system name
// 	// Container name
// 	// Virtual machine name
// 	// Tool/function (Gemini)
// 	Kit string

// 	// func | system | remote
// 	Type string

// 	State State

// 	// func name
// 	Name        string
// 	Description string
// 	Parameters  map[string]any

// 	Body string

// 	//
// 	Config *ToolsConfig
// }

// // ID returns a unique identifier for the tool function,
// // combining the tool kit and function name.
// func (r *ToolFunc) ID() string {
// 	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
// }

type ConnectorConfig struct {
	// mcp | ssh ...
	Proto string `yaml:"proto"`

	// mcp stdin/stdout
	// https://github.com/modelcontextprotocol/servers/tree/main
	// Command string `yaml:"command"`
	// Args    string `yaml:"args"`

	// ssh://user@example.com:2222/user/home
	// git@github.com:owner/repo.git
	// postgres://dbuser:secret@db.example.com:5432/mydb?sslmode=require
	// https://drive.google.com/drive/folders
	// mailto:someone@example.com
	URL string `yaml:"url"`
}
