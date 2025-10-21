package api

import (
	// "encoding/json"
	"html/template"
	// "os"
	"strings"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

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
	}
}

type Flow struct {
	Type        FlowType
	Expression  string
	Concurrency int
	Retry       int
	Actions     []*Action
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

	// set/level
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
	Kit   string        `yaml:"kit"`
	Type  string        `yaml:"type"`
	Tools []*ToolConfig `yaml:"tools"`

	// modelset
	Set      string                  `yaml:"set"`
	Provider string                  `yaml:"provider"`
	BaseUrl  string                  `yaml:"base_url"`
	ApiKey   string                  `yaml:"api_key"`
	Models   map[string]*ModelConfig `yaml:"models"`
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

	Routine *FlowConfig `yaml:"routine"`

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

	//
	Store AssetStore `yaml:"-"`
	// relative to root
	BaseDir string `yaml:"-"`
}

type Instruction struct {
	// Role string `yaml:"role"`
	// TODO add new field
	// Source ? resource/file/cloud...
	// prefix supported: file: resource:
	Content string `yaml:"content"`
	// template or not
	// tpl
	// text/x-go-template
	Type string `yaml:"type"`
}

type FlowType string

const (
	// actions are executed sequentially
	FlowTypeSequence FlowType = "sequence"
	// actions are executed in parallel and the final result will be a list
	FlowTypeParallel FlowType = "parallel"
	// action(s) are executed in a loop with a counter or expression evaluated for each cycle
	FlowTypeLoop FlowType = "loop"
	// one of the actions is selected based on an expression or randomly if no expression is provided
	// expression must evaluate to an integer (zero based).
	FlowTypeChoice FlowType = "choice"
	// The map flow creates a new array populated with the results of calling the action(s)
	// on every element in the input array
	FlowTypeMap FlowType = "map"
	// The reduce flow executes the action(s) on each element of the array, in order,
	// passing in the return value from the calculation on the preceding element.
	// The final result of running the reducer across all elements of the array is returned as a single value.
	// The first time that the flow is run, an initial value is read from the result of the previous agent
	// or empty if the flow is the root agent.
	FlowTypeReduce FlowType = "reduce"
	//
	FlowTypeNest FlowType = "nest"
)

type FlowConfig struct {
	Flow FlowType `yaml:"flow"`
	// // input object key. action input is read from this key. default: result
	// Input string `yaml:"input"`
	// // output object key. action output is saved with this key. default: result
	// Output string `yaml:"output"`
	// go template syntax
	Expression  string   `yaml:"expression"`
	Concurrency int      `yaml:"concurrency"`
	Retry       int      `yaml:"retry"`
	Actions     []string `yaml:"actions"`
}

// type ActionConfig struct {
// 	// Agent string `yaml:"agent"`

// 	// kit:name
// 	Tool  string `yaml:"tool"`
// }

type Resource struct {
	// web resource base url
	// http://localhost:18080/resource
	// https://ai.dhnt.io/resource
	Base string `json:"base"`

	// access token
	Token string `json:"token"`
}
