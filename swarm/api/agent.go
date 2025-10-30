package api

import (
	"html/template"
	"strings"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

// Agent handler
type Handler interface {
	Serve(*Request, *Response) error
}

type State int

func (r State) String() string {
	switch r {
	case StateDefault:
		return "DEFAULT"
	case StateExit:
		return "EXIT"
	case StateTransfer:
		return "TRANSFER"
	case StateInputWait:
		return "INPUT_WAIT"
	}
	return "DEFAULT"
}

func (r State) Equal(s string) bool {
	return strings.ToUpper(s) == r.String()
}

func ParseState(s string) State {
	switch strings.ToUpper(s) {
	case "EXIT":
		return StateExit
	case "TRANSFER":
		return StateTransfer
	case "INPUT_WAIT":
		return StateInputWait
	default:
		return StateDefault
	}
}

const (
	StateDefault State = iota

	StateExit
	StateTransfer
	StateInputWait
)

type TemplateFuncMap = template.FuncMap

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

	Arguments map[string]any

	// LLM adapter
	Adapter string

	//
	Format string

	MaxTurns int
	MaxTime  int

	New        bool
	MaxHistory int
	MaxSpan    int
	Context    string

	LogLevel LogLevel

	//
	Flow *Flow

	//
	Embed []*Agent
}

func (a *Agent) Clone() *Agent {
	return &Agent{
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
		New:         a.New,
		MaxHistory:  a.MaxHistory,
		MaxSpan:     a.MaxSpan,
		Context:     a.Context,
		LogLevel:    a.LogLevel,
		//
		Flow: a.Flow,
		//
		Embed: a.Embed,
	}
}

type Flow struct {
	Type        FlowType
	Expression  string
	Concurrency int
	Retry       int
	Actions     []*Action
	Script      string
}

type Action struct {
	Tool *ToolFunc
}

func (a *Agent) cloneArguments() map[string]any {
	if a.Arguments == nil {
		return nil
	}
	clone := make(map[string]any, len(a.Arguments))
	for k, v := range a.Arguments {
		clone[k] = v
	}
	return clone
}

// agent app config
type AgentsConfig struct {
	// agent app name
	Name string `yaml:"name"`

	// set/level key - not the LLM model
	Model string `yaml:"model"`

	Agents []*AgentConfig `yaml:"agents"`

	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// user message
	Message string `yaml:"message"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory
	// max history: 0 max span: 0
	New        *bool  `yaml:"new,omitempty"`
	MaxHistory int    `yaml:"max_history"`
	MaxSpan    int    `yaml:"max_span"`
	Context    string `yaml:"context"`

	// logging: quiet | informative | verbose
	LogLevel string `yaml:"log_level"`

	// toolkit
	// kit:any
	Kit   string        `yaml:"kit"`
	Type  string        `yaml:"type"`
	Tools []*ToolConfig `yaml:"tools"`

	// model set
	// set/any
	Set    string                  `yaml:"set"`
	Models map[string]*ModelConfig `yaml:"models"`

	// TODO separate?
	// model/tool shared default values
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// api lookup key
	ApiKey string `yaml:"api_key"`
}

type AgentConfig struct {
	Name        string `yaml:"name"`
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	//
	Instruction *Instruction `yaml:"instruction"`

	// set/level
	Model string `yaml:"model"`

	// tools defined in tools config
	// kit:name
	Functions []string `yaml:"functions"`

	Flow *FlowConfig `yaml:"flow"`

	// chat|image|docker/aider oh gptr
	Adapter string `yaml:"adapter"`

	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// user message
	Message string `yaml:"message"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory
	// max history: 0 max span: 0
	New        *bool  `yaml:"new,omitempty"`
	MaxHistory int    `yaml:"max_history"`
	MaxSpan    int    `yaml:"max_span"`
	Context    string `yaml:"context"`

	// logging: quiet | info[rmative] | verbose | trace
	LogLevel string `yaml:"log_level"`

	// agent as tool
	Parameters map[string]any `yaml:"parameters"`

	// default values for parameters
	Arguments map[string]any `yaml:"arguments"`

	// security
	Filters []*IOFilter  `yaml:"filters"`
	Guards  []*ToolGuard `yaml:"guards"`

	// inherit parent agents'
	// message/instruction, model, and tools
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

	// FlowTypeScript delegates control to a shell script using bash script syntax, enabling
	// complex flow control scenarios driven by external scripting logic.
	FlowTypeScript FlowType = "script"
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
