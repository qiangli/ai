package api

import (
	"fmt"
)

type ToolConfig struct {
	Kit string `yaml:"kit"`

	// Internal bool `yaml:"internal"`

	Type string `yaml:"type"`

	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`

	Condition *ToolCondition `yaml:"condition"`

	Body string `yaml:"body"`
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

type ToolsConfig struct {
	// default kit name
	Kit string `yaml:"kit"`

	// Default type: func | system | mcp
	Type string `yaml:"type"`

	// kit config is discarded if true
	Internal bool `yaml:"internal"`

	// system commands used by tools
	Commands []string `yaml:"commands"`

	Tools []*ToolConfig `yaml:"tools"`
}

// ToolDescriptor is a description of a tool function.
type ToolDescriptor struct {
	Name        string
	Description string
	Parameters  map[string]any

	Body string
}

type ToolFunc struct {
	// Namespace:
	//
	// func class
	// Agent name
	// MCP server name
	// Virtual file system name
	// Container name
	// Virtual machine name
	// Tool/function (Gemini)
	Kit string

	// func | system
	Type string

	State State

	// func name
	Name        string
	Description string
	Parameters  map[string]any

	Body string
}

// ID returns a unique identifier for the tool function,
// combining the tool kit and function name.
func (r *ToolFunc) ID() string {
	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
}
