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

	// agent name if this tool references an agent
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
// A string that must match the pattern '^[a-zA-Z0-9_-]+$'."
func (r *ToolFunc) ID() string {
	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
}

// Toolkit configuration
type ToolsConfig struct {
	// kit name specifies a namespace.
	// e.g. but not limited to:
	// class name
	// MCP server name
	// virtual filesystem name
	// container name
	// virtual machine name
	// tool/function (Gemini)
	Kit string `yaml:"kit"`

	// func (server) | system (client) | remote
	Type string `yaml:"type"`

	// provider
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	// Connector *ConnectorConfig `yaml:"connector"`

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

	// filter by match key=values (comma, separated)
	// include all tools that match
	Filter map[string]string `yaml:"filter"`

	// Extra map[string]string `yaml:"extra"`
}

type FuncBody struct {
	Language string `yaml:"language"`
	Code     string `yaml:"code"`
	Url      string `yaml:"url"`
}

// TODO
// condidtion needs to be met for tools to be enabled
type ToolCondition struct {
	// required env list
	Env []string `yaml:"env"`

	// found on PATH
	Lookup *string `yaml:"lookup"`

	// shell required
	Shell map[string]any `yaml:"shell"`
}

type ConnectorConfig struct {
	// mcp | ssh ...
	// Proto string `yaml:"proto"`

	// mcp stdin/stdout
	// https://github.com/modelcontextprotocol/servers/tree/main
	// Command string `yaml:"command"`
	// Args    string `yaml:"args"`

	// ssh://user@example.com:2222/user/home
	// git@github.com:owner/repo.git
	// postgres://dbuser:secret@db.example.com:5432/mydb?sslmode=require
	// https://drive.google.com/drive/folders
	// mailto:someone@example.com

	// optional as of now
	Provider string `yaml:"provider"`

	BaseUrl string `yaml:"base_url"`
	// name of api lookup key
	ApiKey string `yaml:"api_key"`

	// Extra map[string]string `yaml:"extra"`
}

// per tool call vars
type ToolEnv struct {
	// User  string
	Owner string
}

type ToolKit interface {
	Call(context.Context, *Vars, *ToolEnv, *ToolFunc, map[string]any) (any, error)
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
// agent:name
// @name
// @:name
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
	if strings.HasPrefix(s, "@") {
		// <agent:name>
		// @name
		// @name:*
		// @:name
		kit, name = split2(s, ":", "")
		if v := strings.TrimPrefix(kit, "@"); v != "" {
			name = v
		}
		return "agent", name
	}
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
