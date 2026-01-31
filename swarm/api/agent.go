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
	// models - a special arg for supporting multi model fallback.

	// for tool or agent as a tool
	Parameters Parameters `json:"parameters"`

	// templated values.
	// these should not be in the args map (ignored)
	// inherit from embedded ancestors
	Instruction string `json:"instruction"`
	Context     string `json:"context"`
	// not inherited
	Message string `json:"message"`

	// The preferred model to be used by the agent
	Model *Model `json:"model"`

	// The predefined list of functions the agent can call
	Tools []*ToolFunc `json:"-"`

	// inheritance
	Embed []*Agent `json:"-"`

	// LLM adapter
	Adapter string `json:"adapter"`

	// custom actions
	Entrypoint []string `json:"entrypoint"`

	// advices
	Before []string `json:"before"`
	Around []string `json:"around"`
	After  []string `json:"after"`

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

	// resolved from instruction/message/context
	Prompt  string     `json:"prompt"`
	Query   string     `json:"query"`
	History []*Message `json:"history"`

	// resolved from models arg
	Models []*Model `json:"models"`
}

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
	// Type: "agent"

	Name        string `yaml:"name" json:"name"` // sub name without pack
	Display     string `yaml:"display" json:"display"`
	Description string `yaml:"description" json:"description"`

	Environment map[string]any `yaml:"environment" json:"environment"` // global vars
	Arguments   map[string]any `yaml:"arguments" json:"arguments"`

	// agent as tool: "agent"
	Parameters Parameters `yaml:"parameters" json:"parameters"` // agent as tool

	// LLM that generates the output
	Instruction string `yaml:"instruction" json:"instruction"`
	Context     string `yaml:"context" json:"context"`
	Message     string `yaml:"message" json:"message"`
	Model       string `yaml:"model" json:"model"`

	// output destinateion: console, none, file:/
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

	// TODO clarify/finalize
	// inherit from embedded ancestors (agent):
	// + environmen
	//
	// + instruction (prompt)
	// + context (history)
	//
	// + model
	// + functions
	//
	// local scope only (agent/tool):
	// - arguments
	// - parameters
	//
	// - message (query)
	Embed []string `yaml:"embed" json:"embed"`

	// chat|image|docker
	Adapter string `yaml:"adapter" json:"adapter"`

	//
	Entrypoint []string      `yaml:"entrypoint" json:"entrypoint"`
	Advices    *AdviceConfig `yaml:"advices" json:"advices"`

	// runtime
	Config *AppConfig `json:"-"`
}

// only for valid supported fields
// non zero value only
func (c *AgentConfig) ToMap() map[string]any {
	result := make(map[string]any)

	if c.Name != "" {
		result["name"] = c.Name
	}
	if c.Display != "" {
		result["display"] = c.Name
	}

	if c.Model != "" {
		result["model"] = c.Model
	}

	if c.MaxTurns > 0 {
		result["max_turns"] = c.MaxTurns
	}
	if c.MaxTime > 0 {
		result["max_time"] = c.MaxTime
	}
	if c.MaxHistory > 0 {
		result["max_history"] = c.MaxHistory
	}
	if c.MaxSpan > 0 {
		result["max_span"] = c.MaxSpan
	}
	if c.LogLevel != "" {
		result["log_level"] = c.LogLevel
	}

	return result
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
