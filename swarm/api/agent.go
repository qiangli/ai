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

	// Agent sub name
	Name        string `json:"name"`
	Display     string `json:"display"`
	Description string `json:"description"`

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

	// LLM adapter
	Adapter string `json:"adapter"`

	// inheritance
	Embed []*Agent `json:"-"`

	// exported global values
	// Environment map[string]any
	Environment *Environment `json:"environment"`

	// default values
	Arguments Arguments `json:"arguments"`

	Parameters Parameters `json:"parameters"`

	// // model fallback
	// Models map[string]Setlevel `json:"-"`

	// assigned at buildtime/runtime
	Parent   *Agent             `json:"-"`
	Runner   ActionRunner       `json:"-"`
	Shell    ActionRunner       `json:"-"`
	Template *template.Template `json:"-"`

	Config *AppConfig `json:"-"`
}

// // for reusing cached agent
// func (a *Agent) Clone() *Agent {
// 	clone := &Agent{
// 		Pack:        a.Pack,
// 		//
// 		Name:        a.Name,
// 		Display:     a.Display,
// 		Description: a.Description,
// 		//
// 		Instruction: a.Instruction,
// 		Context:     a.Context,
// 		Message:     a.Message,
// 		//
// 		Model:     a.Model,
// 		Tools:     a.Tools,
// 		Adapter:   a.Adapter,
// 		//
// 		Embed:       a.Embed,
// 		//
// 		Environment: a.cloneEnvironment(),
// 		Arguments: a.cloneArguments(),
// 		Parameters: a.Parameters,
// 		//
// 		Models: a.Models,
// 		//
// 		Parent: a.Parent,
// 		Runner:   a.Runner,
// 		Shell:    a.Shell,
// 		Template: a.Template,
// 		//
// 		Config: a.Config,
// 	}

// 	return clone
// }

// func (a *Agent) cloneArguments() Arguments {
// 	if a.Arguments == nil {
// 		return nil
// 	}
// 	return a.Arguments.Clone()
// }

// func (a *Agent) cloneEnvironment() *Environment {
// 	if a.Environment == nil {
// 		return nil
// 	}
// 	return a.Environment.Clone()
// }

type AgentConfig struct {
	Display     string `yaml:"display" json:"display"`
	Description string `yaml:"description" json:"description"`

	// tools defined in tools config
	// kit:name
	Functions []string `yaml:"functions" json:"functions"`

	// Flow *FlowConfig `yaml:"flow" json:"flow"`

	// chat|image|docker/aider oh gptr
	Adapter string `yaml:"adapter" json:"adapter"`

	// // name of custom creator agent for this agent configuration
	// Creator string `yaml:"creator" json:"creator"`

	// default agents config
	// sub name only
	Name      string         `yaml:"name" json:"name"`
	Arguments map[string]any `yaml:"arguments" json:"arguments"`

	//
	Instruction string `yaml:"instruction" json:"instruction"`
	Context     string `yaml:"context" json:"context"`
	Message     string `yaml:"message" json:"message"`

	Model string `yaml:"model" json:"model"`

	//
	MaxTurns   int `yaml:"max_turns" json:"max_turns"`
	MaxTime    int `yaml:"max_time" json:"max_time"`
	MaxHistory int `yaml:"max_history" json:"max_history"`
	MaxSpan    int `yaml:"max_span" json:"max_span"`

	// logging: quiet | info[rmative] | verbose | trace
	LogLevel string `yaml:"log_level" json:"log_level"`

	// agent as tool
	Parameters Parameters `yaml:"parameters" json:"parameters"`

	// agent global vars
	Environment map[string]any `yaml:"environment" json:"environment"`

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

	//
	Entrypoint []string `yaml:"entrypoint" json:"entrypoint"`

	// model fallback
	Models map[string]Setlevel `yaml:"models" json:"models"`

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
