package api

import (
	"context"
	"encoding/json"
	"fmt"

	"strings"
	"time"
)

// TODO revisit real type
// func/web/system/*, mcp, agent, bin, ai???
type ToolType string

const (
	// script/template
	ToolTypeFunc ToolType = "func"

	ToolTypeSystem ToolType = "system"

	ToolTypeWeb ToolType = "web"

	ToolTypeMcp ToolType = "mcp"

	ToolTypeAgent ToolType = "agent"

	ToolTypeAI ToolType = "ai"

	//
	ToolTypeBin ToolType = "bin"

	ToolTypeAlias ToolType = "alias"
)

type ToolFunc struct {
	Type        ToolType       `json:"type"`
	Kit         string         `json:"kit"`         // kit
	Name        string         `json:"name"`        // name
	Description string         `json:"description"` // description
	Parameters  map[string]any `json:"parameters"`  // parameters

	Body *FuncBody `json:"body"` // body

	// agent name if this tool references an agent
	Agent string `json:"agent"`

	//
	State State `json:"state"`

	//
	Provider string `json:"provider"`
	BaseUrl  string `json:"base_url"` // base url
	// name of api key - used to resolve api key/token before tool call
	ApiKey string `json:"api_key"`

	// extra features.
	// e.g labels for mcp for filtering tools
	Extra map[string]any `json:"extra"`

	// default arguments
	Arguments map[string]any `json:"arguments"`
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

type ToolConfig struct {
	Type string `yaml:"type" json:"type"`

	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`

	Parameters map[string]any `yaml:"parameters" json:"parameters"`

	Body *FuncBody `yaml:"body" json:"body"`

	// agent name for agent tool type
	// description/parameters defined here take precedence
	Agent string `yaml:"agent" json:"agent"`

	//
	Provider string `yaml:"provider" json:"provider"`
	BaseUrl  string `yaml:"base_url" json:"base_url"`
	// api lookup key
	ApiKey string `yaml:"api_key" json:"api_key"`

	// filter by match key=values (comma, separated)
	// include all tools that match
	Filter map[string]string `yaml:"filter" json:"filter"`

	//
	Config *AppConfig `json:"-"`
}

type FuncBody struct {
	// A MIME type (now properly called "media type", but also sometimes "content type")
	// is a string sent along with a file indicating the type of the file
	// (describing the content format, for example, a sound file might be labeled audio/ogg,
	// or an image file image/png).
	// https://developer.mozilla.org/en-US/docs/Glossary/MIME_type
	// tpl   text/x-go-template
	// uri   text/uri-list
	// md    text/markdown
	// txt   text/plain
	// go    application/x-go
	// yaml  application/yaml
	// sh    application/x-sh
	// json  application/json
	// js    text/javascript
	MimeType string `yaml:"mime_type" json:"mime_type"`
	Script   string `yaml:"script" json:"script"`
	// Language string `yaml:"language" json:"language"`
	// Code     string `yaml:"code" json:"code"`
	// Url      string `yaml:"url" json:"url"`
}

// // TODO
// // condidtion needs to be met for tools to be enabled
// type ToolCondition struct {
// 	// required env list
// 	Env []string `yaml:"env"`

// 	// found on PATH
// 	Lookup *string `yaml:"lookup"`

// 	// shell required
// 	Shell map[string]any `yaml:"shell"`
// }

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
