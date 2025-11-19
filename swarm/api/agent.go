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

	// system prompt
	Instruction *Instruction
	// user query
	Message string

	RawInput *UserInput

	// The model to be used by the agent
	Model *Model
	// Functions that the agent can call
	Tools []*ToolFunc

	// default values
	Arguments *Arguments

	// LLM adapter
	Adapter string

	//
	Format string

	MaxTurns int
	MaxTime  int

	MaxHistory int
	MaxSpan    int
	Context    string

	LogLevel LogLevel

	//
	Flow *Flow

	Embed []*Agent

	// global values
	// Environment map[string]any
	Environment *Environment

	//
	Runner ActionRunner

	Template *template.Template

	// mu sync.RWMutex

	// // conversation history
	// history []*Message `json:"-"`
	// // initial size of hisotry
	// initLen int `json:"-"`
}

// // Clear messages from history
// func (a *Agent) ClearHistory() {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()
// 	a.history = []*Message{}
// 	a.initLen = 0
// }

// func (a *Agent) InitHistory(messages []*Message) {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()
// 	a.history = messages
// 	a.initLen = len(messages)
// }

// func (a *Agent) GetNewHistory() []*Message {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()
// 	if len(a.history) > a.initLen {
// 		return a.history[a.initLen:]
// 	}
// 	return nil
// }

// // Append messages to history
// func (a *Agent) AddHistory(messages []*Message) {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()
// 	a.history = append(a.history, messages...)
// }

// // Return a copy of all current messages in history
// func (a *Agent) ListHistory() []*Message {
// 	a.mu.RLock()
// 	defer a.mu.RUnlock()
// 	hist := make([]*Message, len(a.history))
// 	copy(hist, a.history)
// 	return hist
// }

// if true, skip historical messages for LLM context
// --new command line flag sets --max-history=0
func (a *Agent) New() bool {
	return a.MaxHistory == 0
}

func (a *Agent) Clone() *Agent {
	clone := &Agent{
		Parent:      a.Parent,
		Owner:       a.Owner,
		Name:        a.Name,
		Display:     a.Display,
		Description: a.Description,
		Instruction: a.Instruction,
		RawInput:    a.RawInput,
		Model:       a.Model,
		Tools:       a.Tools,
		Arguments:   a.cloneArguments(),
		Adapter:     a.Adapter,
		Message:     a.Message,
		Format:      a.Format,
		MaxTurns:    a.MaxTurns,
		MaxTime:     a.MaxTime,
		// New:         a.New,
		MaxHistory: a.MaxHistory,
		MaxSpan:    a.MaxSpan,
		Context:    a.Context,
		LogLevel:   a.LogLevel,
		//
		Flow: a.Flow,
		//
		Embed:       a.Embed,
		Environment: a.Environment.Clone(),
		//
		// history: make([]*Message, len(a.history)),
		Runner: a.Runner,
	}

	// copy(clone.history, a.history)
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

// func (a *Agent) cloneEnvironment() map[string]any {
// 	if a.Environment == nil {
// 		return nil
// 	}
// 	clone := make(map[string]any, len(a.Environment))
// 	maps.Copy(clone, a.Environment)
// 	return clone
// }

// pack config
type AgentsConfig ActionConfig

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
	// context
	// instruction
	// message
	// model
	// tools
	Embed []string `yaml:"embed"`

	//
	Store AssetStore `yaml:"-"`
	// relative to root
	BaseDir string `yaml:"-"`
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

	// FlowTypeReduce applies action(s) sequentially to each element of an input array, accumulating
	// results. It passes the result of each action as input to the next. The process returns a single
	// accumulated value. If at the root, an initial value is sourced from a previous agent or user query.
	FlowTypeReduce FlowType = "reduce"

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
