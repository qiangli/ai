package api

import (
	"fmt"
)

type ToolConfig struct {
	Kit string `yaml:"kit"`

	Internal bool `yaml:"internal"`

	Type        string         `yaml:"type"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`

	Body string `yaml:"body"`
}

type ToolsConfig struct {
	Kit string `yaml:"kit"`

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
	Type string

	// func class
	// Agent name
	// MCP server name
	// Virtual file system name
	// Container name
	// Virtual machine name
	// Tool/function (Gemini)
	Kit string

	// func name
	Name        string
	Description string
	Parameters  map[string]any

	Body string
}

// ID returns a unique identifier for the tool function,
// combining the tool name and function name.
func (r *ToolFunc) ID() string {
	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
}
