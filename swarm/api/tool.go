package api

import (
	"context"
	"fmt"
)

const (
	ToolTypeFunc   = "func"
	ToolTypeSystem = "system"
	ToolTypeWeb    = "web"
	ToolTypeMcp    = "mcp"
)

type ToolCaller func(*Vars, *Agent) func(context.Context, string, map[string]any) (*Result, error)

type ToolFunc struct {
	Type string

	Kit         string
	Name        string
	Description string
	Parameters  map[string]any

	Body string

	//
	State State
	//
	Config *ToolsConfig

	//
	Provider string
	BaseUrl  string
	// name of api key - used to resolve api key/token before tool call
	ApiKey string

	Extra map[string]string
}

// ID returns a unique identifier for the tool,
// combining the tool kit and name.
func (r *ToolFunc) ID() string {
	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
}

// Toolkit configuration
type ToolsConfig struct {
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

	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	Connector *ConnectorConfig `yaml:"connector"`

	// system commands used by tools
	Commands []string `yaml:"commands"`

	Tools []*ToolConfig `yaml:"tools"`
}

type ToolConfig struct {
	Type string `yaml:"type"`

	Name string `yaml:"name"`

	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`

	Body string `yaml:"body"`

	Condition *ToolCondition `yaml:"condition"`

	//
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	Extra map[string]string `yaml:"extra"`
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

	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	Extra map[string]string `yaml:"extra"`
}

type ToolSystem interface {
	Call(context.Context, *Vars, *ToolFunc, map[string]any) (*Result, error)
}
