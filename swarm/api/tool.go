package api

import (
	"context"
	"encoding/json"
	"fmt"
	// "io/fs"
	"strings"
	"time"
)

// TODO real type of string
type ToolType string

const (
	ToolTypeFunc   ToolType = "func"
	ToolTypeSystem ToolType = "system"
	ToolTypeWeb    ToolType = "web"
	ToolTypeMcp    ToolType = "mcp"

	ToolTypeAgent ToolType = "agent"
	ToolTypeAI    ToolType = "ai"
)

type ToolFunc struct {
	Type ToolType

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
	Provider string
	BaseUrl  string
	// name of api key - used to resolve api key/token before tool call
	ApiKey string

	// extra features.
	// e.g labels for mcp for filtering tools
	Extra map[string]any

	// default arguments
	Arguments map[string]any
}

func (r *ToolFunc) Clone() *ToolFunc {
	// Create a new ToolFunc
	clone := *r

	// NOTE Deep copy the Body?
	clone.Body = r.Body

	// Deep copy the Parameters map
	if r.Parameters != nil {
		clone.Parameters = make(map[string]any, len(r.Parameters))
		for k, v := range r.Parameters {
			clone.Parameters[k] = v
		}
	}

	// Deep copy the Extra map
	if r.Extra != nil {
		clone.Extra = make(map[string]any, len(r.Extra))
		for k, v := range r.Extra {
			clone.Extra[k] = v
		}
	}

	return &clone
}

// ID returns a unique identifier for the tool,
// combining the tool kit and name.
// A string that must match the pattern '^[a-zA-Z0-9_-]+$'."
func (r *ToolFunc) ID() string {
	return toolID(r.Kit, r.Name)
}

// Toolkit config
// type ToolsConfig AppConfig

type ToolConfig struct {
	Type string `yaml:"type"`

	Name        string `yaml:"name"`
	Description string `yaml:"description"`

	Parameters map[string]any `yaml:"parameters"`

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
}

// per tool call vars
type ToolEnv struct {
	// User string
	Agent *Agent
	// FS      fs.FS
	// Secrets SecretStore
}

type ToolKit interface {
	Call(context.Context, *Vars, *ToolEnv, *ToolFunc, map[string]any) (any, error)
}

type ToolSystem interface {
	GetKit(key any) (ToolKit, error)
	AddKit(key any, kit ToolKit)
}

// Took Kit and Name
// ^[a-zA-Z0-9_-]+$
type Kitname string

func (r Kitname) String() string {
	return string(r)
}

// agent:
// agent__
func (r Kitname) IsAgent() bool {
	kit, _ := r.Decode()
	return kit == string(ToolTypeAgent)
}

// kit__name
// kit:*
// kit:name
// agent:name
// @name
// @:name
func (r Kitname) Decode() (string, string) {
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
		// kit, name = split2(s, ":", "*")
		kit, name = split2(s, ":", "")
	}
	return kit, name
}

// ^[a-zA-Z0-9_-]+$
func (r Kitname) ID() string {
	kit, name := r.Decode()
	return toolID(kit, name)
}

// TODO all special chars?
// ^[a-zA-Z0-9_-]+$
func tr(s string) string {
	return strings.ReplaceAll(s, "/", "_")
}

func toolID(kit, name string) string {
	// TODO update agent lookup to use ID "agent__pack_sub"
	if kit == "agent" {
		return fmt.Sprintf("%s__%s", kit, name)
	}
	return fmt.Sprintf("%s__%s", kit, tr(name))
}

type ToolGuard struct {
	Agent string
}

type ToolCallEntry struct {
	ID        string         `json:"id"`
	Kit       string         `json:"kit"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`

	Error  error `json:"error"`
	Result any   `json:"result"`

	Timestamp time.Time `json:"timestamp"`
}

func (r *ToolCallEntry) ToJSON() (string, error) {
	bytes, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
