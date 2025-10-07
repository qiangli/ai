package api

import (
	"context"
	"fmt"
	"strings"
)

const (
	ToolTypeFunc   = "func"
	ToolTypeSystem = "system"
	ToolTypeWeb    = "web"
	ToolTypeMcp    = "mcp"
	ToolTypeFaas   = "faas"
	ToolTypeAgent  = "agent"
)

type ToolRunner func(context.Context, string, map[string]any) (*Result, error)

type ToolFunc struct {
	Type string

	Kit         string
	Name        string
	Description string
	Parameters  map[string]any

	Body *FuncBody

	// agent name of agent tool
	Agent string

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

	Body *FuncBody `yaml:"body"`

	Condition *ToolCondition `yaml:"condition"`

	// agent name for agent tool type
	// description/parameters defined here take precedence
	Agent string `yaml:"agent"`

	//
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	Extra map[string]string `yaml:"extra"`
}

type FuncBody struct {
	Language string `yaml:"language"`
	Code     string `yaml:"code"`
	Url      string `yaml:"url"`
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

// // ToolDescriptor is a description of a tool function.
// type ToolDescriptor struct {
// 	Name        string
// 	Description string
// 	Parameters  map[string]any

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

	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	Extra map[string]string `yaml:"extra"`
}

type SecretToken func() (string, error)

type ToolKit interface {
	Call(context.Context, *Vars, SecretToken, *ToolFunc, map[string]any) (any, error)
}

type ToolSystem interface {
	GetKit(key any) (ToolKit, error)
	AddKit(key any, kit ToolKit)
}

type KitName string

func (r KitName) String() string {
	return string(r)
}

// kit__name
// kit:*
// kit:name
func (r KitName) Decode() (string, string) {
	split2 := func(s string, sep string, val string) (string, string) {
		var p1, p2 string
		parts := strings.SplitN(s, sep, 2)
		if len(parts) == 1 {
			p1 = parts[0]
			p2 = val
		} else {
			p1 = parts[0]
			p2 = parts[1]
		}
		return p1, p2
	}

	var kit, name string
	s := string(r)
	if strings.Index(s, "__") > 0 {
		// call time - the name should never be empty
		kit, name = split2(s, "__", "")
	} else {
		// load time
		kit, name = split2(s, ":", "*")
	}
	return kit, name
}

type ToolGuard struct {
	Agent string
}
