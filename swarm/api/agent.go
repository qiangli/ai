package api

import (
	"encoding/json"
	"html/template"
	"os"
	"strings"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

// type AgentCreator func(*Vars, *Request) (*Agent, error)

type Handler interface {
	Serve(*Request, *Response) error
}

// type AgentHandler func(*Vars, *Agent) Handler

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

// @[owner:]pack[/sub]
// // @<[owner:]agent>
// owner, agent := splitOwnerAgent(req.Agent)
// // agent: pack[/sub]
// pack, sub := split2(agent, "/", "")
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

	Instruction *Instruction

	RawInput *UserInput

	// The model to be used by the agent
	Model *Model
	// Functions that the agent can call
	Tools []*ToolFunc

	Arguments map[string]any

	Dependencies []string
	Entrypoint   Entrypoint

	// advices
	BeforeAdvice Advice
	AfterAdvice  Advice
	AroundAdvice Advice

	// LLM adapter
	Adapter string

	//
	MaxTurns int
	MaxTime  int
	//
	Message string
	Format  string
	New     bool
	// MaxHistory int
	// MaxSpan    int
	LogLevel LogLevel

	//
	Config *AgentsConfig
}

// ... existing code ...

func (a *Agent) Clone() *Agent {
	return &Agent{
		Owner:        a.Owner,
		Name:         a.Name,
		Display:      a.Display,
		Description:  a.Description,
		Instruction:  a.Instruction,
		RawInput:     a.RawInput,
		Model:        a.Model,
		Tools:        a.Tools,
		Arguments:    a.cloneArguments(),
		Dependencies: a.Dependencies,
		Entrypoint:   a.Entrypoint,
		BeforeAdvice: a.BeforeAdvice,
		AfterAdvice:  a.AfterAdvice,
		AroundAdvice: a.AroundAdvice,
		Adapter:      a.Adapter,
		MaxTurns:     a.MaxTurns,
		MaxTime:      a.MaxTime,
		Message:      a.Message,
		Format:       a.Format,
		New:          a.New,
		LogLevel:     a.LogLevel,
		//
		Config: a.Config,
	}
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

	// Active bool `yaml:"active"`

	// [alias/]level
	Model string `yaml:"model"`

	Agents []*AgentConfig `yaml:"agents"`
	Tools  []*ToolConfig  `yaml:"tools"`
	Models []*ModelConfig `yaml:"models"`

	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// experimental

	// user message
	Message string `yaml:"message"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory
	// max history: 0 max span: 0
	New *bool `yaml:"new,omitempty"`
	// MaxHistory int `yaml:"max_history"`
	// MaxSpan    int `yaml:"max_span"`

	// logging: quiet | informative | verbose
	LogLevel string `yaml:"log_level"`
}

type AgentConfig struct {
	Name        string `yaml:"name"`
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	//
	Instruction *Instruction `yaml:"instruction"`

	// [alias/]level
	Model string `yaml:"model"`

	// tools defined in tools config
	// kit:name
	Functions []string `yaml:"functions"`

	Dependencies []string `yaml:"dependencies"`

	Entrypoint string `yaml:"entrypoint"`

	Advices *AdviceConfig `yaml:"advices"`

	// chat|image-get|docker/aider oh gptr
	Adapter string `yaml:"adapter"`

	//
	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// experimental

	// user message
	Message string `yaml:"message"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory
	// max history: 0 max span: 0
	New *bool `yaml:"new,omitempty"`
	// MaxHistory int `yaml:"max_history"`
	// MaxSpan    int `yaml:"max_span"`

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
	Role string `yaml:"role"`
	// TODO add new field
	// Source ? resource/file/cloud...
	// prefix supported: file: resource:
	Content string `yaml:"content"`
	// template or not
	// tpl
	Type string `yaml:"type"`
}

type AdviceConfig struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Around string `yaml:"around"`
}

// Swarm Agents can call functions directly.
// Function should usually return a string values.
// If a function returns an Agent, execution will be transferred to that Agent.
// type Function = func(context.Context, *Vars, string, map[string]any) (*Result, error)

type Advice func(*Vars, *Request, *Response, Advice) error

type Entrypoint func(*Vars, *Agent, *UserInput) error

type AgentResource struct {
	// web resource base url
	// http://localhost:18080/resource/

	Resources []*Resource `json:"resources"`
}

type Resource struct {
	// web resource base url
	// http://localhost:18080/resource
	// https://ai.dhnt.io/resource
	Base string `json:"base"`

	// access token
	Token string `json:"token"`
}

func LoadAgentResource(p string) (*AgentResource, error) {
	var ar AgentResource
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, err
	}
	return &ar, nil
}
