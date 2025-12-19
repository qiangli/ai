package api

import (
	"context"
	"strings"
	"text/template"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

// @pack[/sub]
// agent:pack[/sub]
// @*
// ^[a-zA-Z0-9_-]+$
type Packname string

func (r Packname) String() string {
	return string(r)
}

// @pack[/sub]
// agent:pack[/sub]
func (r Packname) Decode() (string, string) {
	s := strings.ToLower(string(r))
	s = strings.TrimPrefix(s, "@")
	s = strings.TrimPrefix(s, "agent:")
	parts := strings.SplitN(s, "/", 2)

	var pack = parts[0]
	var sub string
	if len(parts) > 1 {
		sub = parts[1]
	}
	// entry
	if sub == pack {
		sub = ""
	}
	return pack, sub
}

// return the normalized name.
// pack or pack/sub after removing any prefix: @ agent: or /
func (r Packname) Encode() string {
	pack, sub := r.Decode()
	if sub == "" {
		return pack
	}
	return pack + "/" + sub
}

func (r Packname) Equal(s string) bool {
	x, y := r.Decode()
	x2, y2 := Packname(s).Decode()
	return x == x2 && y == y2
}

type Creator func(context.Context, string) (*Agent, error)

type Agent struct {
	Parent *Agent `json:"parent"`

	// Package
	Pack string `json:"pack"`

	// The name of the agent.
	Name        string `json:"name"`
	Display     string `json:"display"`
	Description string `json:"description"`

	//
	Instruction string `json:"instruction"`
	Context     string `json:"context"`
	Message     string `json:"message"`

	// The model to be used by the agent
	Model *Model `json:"model"`
	// Functions that the agent can call
	Tools []*ToolFunc `json:"tools"`

	// // default values
	Arguments *Arguments `json:"arguments"`

	// LLM adapter
	Adapter string `json:"adapter"`

	//
	Flow *Flow `json:"flow"`

	Embed []*Agent `json:"embed"`

	// global values
	// Environment map[string]any
	Environment *Environment `json:"environment"`

	//
	Runner   ActionRunner       `json:"-"`
	Shell    ActionRunner       `json:"-"`
	Creator  Creator            `json:"-"`
	Template *template.Template `json:"-"`
	Config   *AppConfig         `json:"-"`
}

// for reusing cached agent
func (a *Agent) Clone() *Agent {
	clone := &Agent{
		Parent: a.Parent,
		// Owner:       a.Owner,
		Name:        a.Name,
		Display:     a.Display,
		Description: a.Description,

		Model:     a.Model,
		Tools:     a.Tools,
		Arguments: a.cloneArguments(),
		Adapter:   a.Adapter,

		//
		Flow: a.Flow,
		//
		Embed:       a.Embed,
		Environment: a.cloneEnvironment(),
		//
		Runner: a.Runner,
	}

	return clone
}

type Flow struct {
	Type       FlowType  `json:"type"`
	Expression string    `json:"expression"`
	Actions    []*Action `json:"actions"`
	Script     string    `json:"script"`
}

func (a *Agent) cloneArguments() *Arguments {
	if a.Arguments == nil {
		return nil
	}
	return a.Arguments.Clone()
}

func (a *Agent) cloneEnvironment() *Environment {
	if a.Environment == nil {
		return nil
	}
	return a.Environment.Clone()
}

type AgentConfig struct {
	Display     string `yaml:"display" json:"display"`
	Description string `yaml:"description" json:"description"`

	// tools defined in tools config
	// kit:name
	Functions []string `yaml:"functions" json:"functions"`

	Flow *FlowConfig `yaml:"flow" json:"flow"`

	// chat|image|docker/aider oh gptr
	Adapter string `yaml:"adapter" json:"adapter"`

	// name of custom creator agent for this agent configuration
	Creator string `yaml:"creator" json:"creator"`

	// default agents config
	Name      string         `yaml:"name" json:"name"`
	Arguments map[string]any `yaml:"arguments" json:"arguments"`

	Instruction string `yaml:"instruction" json:"instruction"`
	Context     string `yaml:"context" json:"context"`
	Message     string `yaml:"message" json:"message"`

	Model string `yaml:"model" json:"model"`

	//
	MaxTurns int `yaml:"max_turns" json:"max_turns"`
	MaxTime  int `yaml:"max_time" json:"max_time"`

	// output format: json | text
	Format string `yaml:"format" json:"format"`

	// memory
	// max history: 0 max span: 0
	// New        *bool  `yaml:"new,omitempty"`
	MaxHistory int `yaml:"max_history" json:"max_history"`
	MaxSpan    int `yaml:"max_span" json:"max_span"`

	// logging: quiet | info[rmative] | verbose | trace
	LogLevel string `yaml:"log_level" json:"log_level"`

	// agent as tool
	Parameters map[string]any `yaml:"parameters" json:"parameters"`

	// agent global vars
	Environment map[string]any `yaml:"environment" json:"environment"`

	// // security
	// Filters []*IOFilter  `yaml:"filters"`
	// Guards  []*ToolGuard `yaml:"guards"`

	// inherit from embedded parent:
	// + environment
	// + instruction
	// + functions
	// local scope:
	// - arguments
	// - context
	// - message
	// - model
	// - flow/actions
	Embed []string `yaml:"embed" json:"embed"`

	//
	Store AssetStore `yaml:"-" json:"-"`
	// relative to root
	BaseDir string `yaml:"-" json:"-"`

	Config *AppConfig `json:"-"`
}

func (ac *AgentConfig) ToMap() map[string]any {
	result := make(map[string]any)
	if ac.Name != "" {
		result["name"] = ac.Name
	}
	if ac.Message != "" {
		result["message"] = ac.Message
	}
	if ac.Instruction != "" {
		result["instruction"] = ac.Instruction
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
	if ac.Format != "" {
		result["format"] = ac.Format
	}
	if ac.MaxHistory > 0 {
		result["max_history"] = ac.MaxHistory
	}
	if ac.MaxSpan > 0 {
		result["max_span"] = ac.MaxSpan
	}
	if ac.Context != "" {
		result["context"] = ac.Context
	}
	if ac.LogLevel != "" {
		result["log_level"] = ac.LogLevel
	}

	return result
}

type FlowType string

const (
	// FlowTypeSequence executes actions one after another, where each
	// subsequent action uses the previous action's response as input.
	FlowTypeSequence FlowType = "sequence"

	// FlowTypeChoice selects and executes a single action based on an evaluated expression.
	// If no expression is provided, an action is chosen randomly. The expression must evaluate
	// to an integer that selects the action index, starting from zero.
	FlowTypeChoice FlowType = "choice"

	// FlowTypeParallel executes actions simultaneously, returning the combined results as a list.
	// This allows for concurrent processing of independent actions.
	FlowTypeParallel FlowType = "parallel"

	// FlowTypeMap applies specified action(s) to each element in the input array, creating a new
	// array populated with the results.
	FlowTypeMap FlowType = "map"

	// // FlowTypeLoop executes actions repetitively in a loop. The loop can use a counter or
	// // evaluate an expression for each iteration, allowing for repeated execution with varying
	// // parameters or conditions.
	// FlowTypeLoop FlowType = "loop"

	// // FlowTypeReduce applies action(s) sequentially to each element of an input array, accumulating
	// // results. It passes the result of each action as input to the next. The process returns a single
	// // accumulated value. If at the root, an initial value is sourced from a previous agent or user query.
	// FlowTypeReduce FlowType = "reduce"

	// FlowTypeShell delegates control to a shell script using bash script syntax, enabling
	// complex flow control scenarios driven by external scripting logic.
	FlowTypeShell FlowType = "shell"
)

type FlowConfig struct {
	Type FlowType `yaml:"type" json:"type"`

	// agent/tool list for non script flow
	Actions []string `yaml:"actions" json:"actions"`

	// content of the script for flow type: script
	Script string `yaml:"script" json:"script"`
}

type Resource struct {
	// web resource base url
	// http://localhost:18080/resource
	// https://ai.dhnt.io/resource
	Base string `json:"base"`

	// access token
	Token string `json:"token"`
}
