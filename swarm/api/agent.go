package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"
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

type Agent struct {
	Owner string

	// The name of the agent.
	Name        string
	Display     string
	Description string

	// // The role of the agent. default is "system"
	// Role string

	Instruction *Instruction

	RawInput *UserInput

	// The model to be used by the agent
	Model *Model

	// Functions that the agent can call
	Tools []*ToolFunc

	// model aliases to used used
	// Models []*Model

	Dependencies []string

	Entrypoint Entrypoint

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
	New bool `yaml:"new"`
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

	// // model alias defined in models config
	// Models string `yaml:"models"`

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
	New bool `yaml:"new"`
	// MaxHistory int `yaml:"max_history"`
	// MaxSpan    int `yaml:"max_span"`

	// logging: quiet | info[rmative] | verbose | trace
	LogLevel string `yaml:"log_level"`

	// agent as tool
	Parameters map[string]any `yaml:"parameters"`

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

type Request struct {
	// The name/command of the active agent
	Agent string
	// Command string

	Messages []*Message

	RawInput *UserInput

	// ctx is either the client or server context. It should only
	// be modified via copying the whole Request using Clone or WithContext.
	// It is unexported to prevent people from using Context wrong
	// and mutating the contexts held by callers of the same request.
	ctx context.Context
}

func NewRequest(ctx context.Context, agent string, input *UserInput) *Request {
	return &Request{
		ctx:      ctx,
		Agent:    agent,
		RawInput: input,
	}
}

// Context returns the request's context. To change the context, use
// [Request.Clone] or [Request.WithContext].
//
// The returned context is always non-nil; it defaults to the
// background context.
//
// For outgoing client requests, the context controls cancellation.
//
// For incoming server requests, the context is canceled when the
// client's connection closes, the request is canceled (with HTTP/2),
// or when the ServeHTTP method returns.
func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

// WithContext returns a shallow copy of r with its context changed
// to ctx. The provided ctx must be non-nil.
//
// For outgoing client request, the context controls the entire
// lifetime of a request and its response: obtaining a connection,
// sending the request, and reading the response headers and body.
//
// To create a new request with a context, use [NewRequest].
// To make a deep copy of a request with a new context, use [Request.Clone].
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.ctx = ctx
	return r2
}

// Clone returns a shallow copy of r with its context changed to ctx.
// The provided ctx must be non-nil.
//
// Clone only makes a shallow copy of the Body field.
//
// For an outgoing client request, the context controls the entire
// lifetime of a request and its response: obtaining a connection,
// sending the request, and reading the response headers and body.
func (r *Request) Clone(ctx context.Context) *Request {
	if ctx == nil {
		panic("nil context")
	}
	r2 := new(Request)
	*r2 = *r
	r2.ctx = ctx
	r2.Agent = r.Agent
	r2.RawInput = r.RawInput

	return r2
}

type Response struct {
	// A list of message objects generated during the conversation
	// with a sender field indicating which Agent the message originated from.
	Messages []*Message

	// The last agent to handle a message
	Agent *Agent

	Result *Result
}

type Message struct {
	ID      string    `json:"id"`
	ChatID  string    `json:"chatId"`
	Created time.Time `json:"created"`

	// data
	ContentType string `json:"contentType"`
	Content     string `json:"content"`

	Role string `json:"role"`

	// agent name
	Sender string `json:"sender"`
}

// Result encapsulates the possible return values for agent/function.
type Result struct {
	// The result value as a string
	Value string

	// Tool call
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/MIME_types
	MimeType string
	Message  string

	// The agent state
	State State

	// The agent name to transfer to for StateTransfer
	NextAgent string
}

func (r *Result) String() string {
	var sb strings.Builder
	if r.State != StateDefault {
		sb.WriteString(r.State.String())
	}
	if r.NextAgent != "" {
		sb.WriteString(fmt.Sprintf(" %s\n", r.NextAgent))
	}
	if r.Value != "" {
		sb.WriteString(fmt.Sprintf(" %s\n", r.Value))
	}
	return strings.TrimSpace(sb.String())
}

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
