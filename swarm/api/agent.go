package api

import (
	"fmt"
	"strings"
	"text/template"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	// RoleTool      = "tool"
)

// @pack[/sub]
// agent:pack[/sub]
// @*
// name: ^[a-z0-9_:/]+$
// id: ^[a-z0-9_]+$
type Packname string

func NewPackname(pack, sub string) Packname {
	return Packname(pack + "/" + sub).Clean()
}

func (r Packname) String() string {
	return string(r)
}

// Return valid id for the agent suitable for use with external systems, e.g. LLM providers.
// agent:pack/sub is converted to agent__pack__sub
// ^[a-z0-9_]+$
func (r Packname) ID() string {
	pack, sub := r.Decode()
	return fmt.Sprintf("agent__%s__%s", pack, sub)
}

// return normalized pack/sub after cleaning. sub is the same as pack if empty
func (r Packname) Decode() (string, string) {

	s := r.Clean()
	parts := strings.SplitN(string(s), "/", 2)
	return parts[0], parts[1]
}

// Return a normalized version: pack/sub
// after removing slash command char '/', prefix and suffix
// [/]@pack[/sub]
// [/]agent:pack[/sub]
// ^[a-z0-9_/]+$
func (r Packname) Clean() Packname {
	s := strings.ToLower(string(r))
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimPrefix(s, "@")
	s = strings.TrimPrefix(s, "agent:")
	//
	s = strings.TrimSuffix(s, ",")

	// convert back __ to / if any
	s = strings.ReplaceAll(s, "__", "/")

	parts := strings.SplitN(s, "/", 2)
	var pack = parts[0]
	var sub string
	if len(parts) > 1 {
		sub = parts[1]
	}
	if sub == "" {
		sub = pack
	}

	return Packname(pack + "/" + sub)
}

func (r Packname) Equal(s string) bool {
	x1 := r.Clean()
	x2 := Packname(s).Clean()
	return x1 == x2
}

type Agent struct {
	// Package name
	Pack string `json:"pack"`

	Name        string `json:"name"` // Agent sub name
	Display     string `json:"display"`
	Description string `json:"description"`

	// exported global values
	// Environment map[string]any
	Environment *Environment `json:"environment"`

	// default values
	Arguments Arguments `json:"arguments"`

	// agent as tool
	Parameters Parameters `json:"parameters"`

	// templated values
	// these should not be in the args map
	Instruction string `json:"instruction"`
	Context     string `json:"context"`
	Message     string `json:"message"`

	// The model to be used by the agent
	// TODO map or list for supporting multi features
	Model *Model `json:"model"`

	// Functions that the agent can call
	Tools []*ToolFunc `json:"tools"`

	// inheritance
	Embed []*Agent `json:"-"`

	// LLM adapter
	Adapter string `json:"adapter"`

	// assigned at buildtime/runtime
	Parent *Agent `json:"-"`
	//
	Runner   ActionRunner       `json:"-"`
	Shell    ActionRunner       `json:"-"`
	Template *template.Template `json:"-"`

	Config *AppConfig `json:"-"`

	// LLM
	// get api token for LLM model
	Token func() string `json:"-"`

	Prompt  string     `json:"prompt"`
	Query   string     `json:"query"`
	History []*Message `json:"history"`

	Models []*Model `json:"models"`
}

// func (r *Agent) MaxTurns() int {
// 	if r.Arguments == nil {
// 		return 0
// 	}
// 	return r.Arguments.GetInt("max_turns")
// }

// func (r *Agent) SetMaxTurns(max int) *Agent {
// 	if r.Arguments == nil {
// 		r.Arguments = NewArguments()
// 	}
// 	r.Arguments.SetArg("max_turns", max)
// 	return r
// }

func (r *Agent) MemOption() *MemOption {
	var o MemOption
	if r.Arguments == nil {
		return &o
	}
	o.MaxHistory = r.Arguments.GetInt("max_history")
	o.MaxSpan = r.Arguments.GetInt("max_span")
	o.Offset = r.Arguments.GetInt("offset")
	o.Roles = r.Arguments.GetStringSlice("roles")

	return &o
}

type AgentConfig struct {
	Name        string `yaml:"name" json:"name"` // sub name without pack
	Display     string `yaml:"display" json:"display"`
	Description string `yaml:"description" json:"description"`

	Environment map[string]any `yaml:"environment" json:"environment"` // global vars
	Arguments   map[string]any `yaml:"arguments" json:"arguments"`
	Parameters  Parameters     `yaml:"parameters" json:"parameters"` // agent as tool

	//
	Instruction string `yaml:"instruction" json:"instruction"`
	Context     string `yaml:"context" json:"context"`
	Message     string `yaml:"message" json:"message"`

	Model string `yaml:"model" json:"model"`

	Output string `yaml:"output" json:"output"`

	// tools defined in tools config
	// kit:name | agent:pack/sub
	Functions []string `yaml:"functions" json:"functions"`

	//
	MaxTurns   int `yaml:"max_turns" json:"max_turns"`
	MaxTime    int `yaml:"max_time" json:"max_time"`
	MaxHistory int `yaml:"max_history" json:"max_history"`
	MaxSpan    int `yaml:"max_span" json:"max_span"`

	// logging: quiet | info[rmative] | verbose | trace
	LogLevel string `yaml:"log_level" json:"log_level"`

	// TODO clarify
	// inherit from embedded parent:
	// + environment
	// + instruction
	// + context
	//
	// + model
	// + functions
	// local scope:
	// - arguments
	// - message
	// - parameters
	Embed []string `yaml:"embed" json:"embed"`

	// chat|image|docker
	Adapter string `yaml:"adapter" json:"adapter"`

	//
	Entrypoint []string      `yaml:"entrypoint" json:"entrypoint"`
	Advices    *AdviceConfig `yaml:"advices" json:"advices"`

	Config *AppConfig `json:"-"`
}

func (ac *AgentConfig) ToMap() map[string]any {
	result := make(map[string]any)

	if ac.Name != "" {
		result["name"] = ac.Name
	}

	if ac.Model != "" {
		result["model"] = ac.Model
	}
	if ac.MaxTurns > 0 {
		result["max_turns"] = ac.MaxTurns
	}
	if ac.MaxTime > 0 {
		result["max_time"] = ac.MaxTime
	}
	if ac.MaxHistory > 0 {
		result["max_history"] = ac.MaxHistory
	}
	if ac.MaxSpan > 0 {
		result["max_span"] = ac.MaxSpan
	}
	if ac.LogLevel != "" {
		result["log_level"] = ac.LogLevel
	}

	return result
}

type Advice struct {
	Before []*ToolFunc `json:"-"`
	Around []*ToolFunc `json:"-"`
	After  []*ToolFunc `json:"-"`
}

type AdviceConfig struct {
	Before []string `yaml:"before" json:"before"`
	Around []string `yaml:"around" json:"around"`
	After  []string `yaml:"after" json:"after"`
}

type FlowType string

const (
	// FlowTypeSequence executes actions one after another, where each
	// subsequent action uses the previous action's response as input.
	FlowTypeSequence FlowType = "sequence"

	// FlowTypeChain Executes a series of actions consecutively, similar to the `sequence` flow.
	// The chain returns the result or error from the final action executed.
	// Chain actions ["a1", "a2", "a3", ...] translate to nested function calls: a1(a2(a3(...))).
	// Each action in the chain should accept a sub-action through the "action" parameter.
	FlowTypeChain FlowType = "chain"

	// FlowTypeChoice randomly selects and executes a single action.
	FlowTypeChoice FlowType = "choice"

	// FlowTypeParallel executes actions simultaneously, returning the combined results as a list.
	// This allows for concurrent processing of independent actions.
	FlowTypeParallel FlowType = "parallel"

	// // FlowTypeMap applies specified action(s) to each element in the input array, creating a new
	// // array populated with the results.
	// FlowTypeMap FlowType = "map"

	// FlowTypeLoop executes actions repetitively in a loop. The loop runs indefinitely or can use a counter.
	FlowTypeLoop FlowType = "loop"

	// Fallback executes actions in sequence. Return the result of the first successfully executed action, or produce an error from the final action if all actions fail.
	FlowTypeFallback FlowType = "fallback"

	// // FlowTypeReduce applies action(s) sequentially to each element of an input array, accumulating
	// // results. It passes the result of each action as input to the next. The process returns a single
	// // accumulated value. If at the root, an initial value is sourced from a previous agent or user query.
	// FlowTypeReduce FlowType = "reduce"

	// FlowTypeShell delegates control to a shell script using bash script syntax, enabling
	// complex flow control scenarios driven by external scripting logic.
	// FlowTypeShell FlowType = "shell"
)

type Resource struct {
	// web resource base url
	// http://localhost:18080/resource
	// https://ai.dhnt.io/resource
	Base string `json:"base"`

	// access token
	Token string `json:"token"`
}
