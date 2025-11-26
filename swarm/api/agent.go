package api

import (
	"strings"
	"text/template"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

// [@][owner:]pack[/sub]
// @[owner:]<agent>
// agent: pack[/sub]
// @any
// @*
type AgentName string

func (a AgentName) String() string {
	return string(a)
}

// [@][owner:]pack[/sub]
func (a AgentName) Decode() (owner, pack, sub string) {
	// @[owner:]agent
	// agent: pack[/sub]
	s := strings.TrimPrefix(string(a), "@")
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		owner = parts[0]
		parts = strings.SplitN(parts[1], "/", 2)
	} else {
		parts = strings.SplitN(parts[0], "/", 2)
	}

	pack = parts[0]
	if len(parts) > 1 {
		sub = parts[1]
	}
	// default sub: "", "<pack>"
	// pack
	// pack/pack
	if sub == pack {
		sub = ""
	}
	return owner, pack, sub
}

func (a AgentName) Equal(s string) bool {
	x, y, z := a.Decode()
	x2, y2, z2 := AgentName(s).Decode()
	return x == x2 && y == y2 && z == z2
}

type Agent struct {
	Parent *Agent

	Owner string

	// The name of the agent.
	Name        string
	Display     string
	Description string

	// // system prompt from config
	// Instruction *Instruction

	// user query from config
	// Message string

	// RawInput *UserInput

	// The model to be used by the agent
	Model *Model
	// Functions that the agent can call
	Tools []*ToolFunc

	// default values
	Arguments *Arguments

	// LLM adapter
	Adapter string

	//
	Flow *Flow

	Embed []*Agent

	// global values
	// Environment map[string]any
	Environment *Environment

	//
	Runner ActionRunner

	Template *template.Template
}

func (a *Agent) Message() string {
	return a.Arguments.GetString("message")
}

func (a *Agent) SetMessage(s string) {
	a.Arguments.Set("message", s)
}

func (a *Agent) Instruction() string {
	return a.Arguments.GetString("instruction")
}

func (a *Agent) SetInstruction(s string) {
	a.Arguments.Set("instruction", s)
}

// if true, skip historical messages for LLM context
// --new command line flag sets --max-history=0
func (a *Agent) New() bool {
	return a.Arguments.GetInt("max_history") == 0
}

func (a *Agent) Clone() *Agent {
	clone := &Agent{
		Parent:      a.Parent,
		Owner:       a.Owner,
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
		Environment: a.Environment.Clone(),
		//
		Runner: a.Runner,
	}

	return clone
}

type Flow struct {
	Type        FlowType
	Expression  string
	Concurrency int
	Retry       int
	Actions     []*Action
	Script      string
}

func (a *Agent) cloneArguments() *Arguments {
	if a.Arguments == nil {
		return nil
	}
	return a.Arguments.Clone()
}

// pack config
type AgentsConfig AppConfig

func (ac *AgentsConfig) ToMap() map[string]any {
	result := make(map[string]any)

	if ac.Kit != "" {
		result["kit"] = ac.Kit
	}
	if ac.Type != "" {
		result["type"] = ac.Type
	}
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

type AgentConfig struct {
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	// tools defined in tools config
	// kit:name
	Functions []string `yaml:"functions"`

	Flow *FlowConfig `yaml:"flow"`

	// chat|image|docker/aider oh gptr
	Adapter string `yaml:"adapter"`

	// name of custom creator agent for this agent configuration
	Creator string `yaml:"creator"`

	// middleware chain
	Chain *ChainConfig `yaml:"chain"`

	// default agents config
	Name      string         `yaml:"name"`
	Arguments map[string]any `yaml:"arguments"`

	Message string `yaml:"message"`

	Instruction *Instruction `yaml:"instruction"`

	Model string `yaml:"model"`

	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory
	// max history: 0 max span: 0
	// New        *bool  `yaml:"new,omitempty"`
	MaxHistory int    `yaml:"max_history"`
	MaxSpan    int    `yaml:"max_span"`
	Context    string `yaml:"context"`

	// logging: quiet | info[rmative] | verbose | trace
	LogLevel string `yaml:"log_level"`

	// agent as tool
	Parameters map[string]any `yaml:"parameters"`

	// agent global vars
	Environment map[string]any `yaml:"environment"`

	// security
	Filters []*IOFilter  `yaml:"filters"`
	Guards  []*ToolGuard `yaml:"guards"`

	// inherit from parent:
	// environment
	// instruction
	// tools
	Embed []string `yaml:"embed"`

	//
	Store AssetStore `yaml:"-"`
	// relative to root
	BaseDir string `yaml:"-"`
}

func (ac *AgentConfig) ToMap() map[string]any {
	result := make(map[string]any)

	if ac.Name != "" {
		result["name"] = ac.Name
	}
	if ac.Message != "" {
		result["message"] = ac.Message
	}
	if ac.Instruction != nil && ac.Instruction.Content != "" {
		result["instruction"] = ac.Instruction.Content
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

type Instruction struct {
	// prefix supported: file: resource:
	// #! [--mime-type=text/x-go-template]
	Content string `yaml:"content"`

	// content type
	// text/x-go-template
	Type string `yaml:"type"`
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

	// FlowTypeLoop executes actions repetitively in a loop. The loop can use a counter or
	// evaluate an expression for each iteration, allowing for repeated execution with varying
	// parameters or conditions.
	FlowTypeLoop FlowType = "loop"

	// // FlowTypeReduce applies action(s) sequentially to each element of an input array, accumulating
	// // results. It passes the result of each action as input to the next. The process returns a single
	// // accumulated value. If at the root, an initial value is sourced from a previous agent or user query.
	// FlowTypeReduce FlowType = "reduce"

	// FlowTypeShell delegates control to a shell script using bash script syntax, enabling
	// complex flow control scenarios driven by external scripting logic.
	FlowTypeShell FlowType = "shell"
)

type FlowConfig struct {
	Type FlowType `yaml:"type"`

	// go template syntax
	Expression  string `yaml:"expression"`
	Concurrency int    `yaml:"concurrency"`
	Retry       int    `yaml:"retry"`

	// agent/tool list for non script flow
	Actions []string `yaml:"actions"`

	// content of the script for flow type: script
	Script string `yaml:"script"`
}

type Resource struct {
	// web resource base url
	// http://localhost:18080/resource
	// https://ai.dhnt.io/resource
	Base string `json:"base"`

	// access token
	Token string `json:"token"`
}

type ChainConfig struct {
	// action list
	Middleware []string `yaml:"middelware"`
}
