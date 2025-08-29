package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api/model"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

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
	// The name of the agent.
	Name    string
	Display string
	// Description string

	// The model to be used by the agent
	Model *model.Model

	// The role of the agent. default is "system"
	Role string

	// Instructions for the agent, can be a string or a callable returning a string
	// Instruction string
	// InstructionType string // file ext
	// Instruction *InstructionConfig

	RawInput *UserInput

	// Functions that the agent can call
	Tools []*ToolFunc

	Entrypoint Entrypoint

	Dependencies []*Agent

	// advices
	BeforeAdvice Advice
	AfterAdvice  Advice
	AroundAdvice Advice

	//
	MaxTurns int
	MaxTime  int

	//
	ResourceMap string

	//
	// Vars *Vars

	Config *AgentConfig
}

type AgentsConfig struct {
	// agent group name
	Name string `yaml:"name"`

	Internal bool `yaml:"internal"`

	Agents    []*AgentConfig    `yaml:"agents"`
	Functions []*FunctionConfig `yaml:"functions"`
	Models    []*ModelConfig    `yaml:"models"`

	MaxTurns int `yaml:"maxTurns"`
	MaxTime  int `yaml:"maxTime"`

	// BaseDir string `yaml:"-"`
	// Source  string `yaml:"-"`
}

type AgentConfig struct {
	Name        string `yaml:"name"`
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	Internal bool   `yaml:"internal"`
	State    string `yaml:"state"`

	//
	Instruction *InstructionConfig `yaml:"instruction"`

	Model string `yaml:"model"`

	Entrypoint string `yaml:"entrypoint"`

	Functions []string `yaml:"functions"`

	Dependencies []string `yaml:"dependencies"`

	Advices *AdviceConfig `yaml:"advices"`

	Store AssetStore `yaml:"-"`
	// relative to root
	BaseDir string `yaml:"-"`
}

type InstructionConfig struct {
	Role string `yaml:"role"`
	// TODO add new field
	// Source ? resource/file/cloud...
	// prefix supported: file: resource:
	Content string `yaml:"content"`
	// template or not
	// tpl
	Type string `yaml:"type"`
}

type FunctionConfig struct {
	Label   string `yaml:"label"`
	Service string `yaml:"service"`

	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`
}

type ModelConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Model       string `yaml:"model"`
	BaseUrl     string `yaml:"baseUrl"`
	ApiKey      string `yaml:"apiKey"`
	External    bool   `yaml:"external"`
}

type AdviceConfig struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Around string `yaml:"around"`
}

// Swarm Agents can call functions directly.
// Function should usually return a string values.
// If a function returns an Agent, execution will be transferred to that Agent.
type Function = func(context.Context, *Vars, string, map[string]any) (*Result, error)

type Advice func(*Vars, *Request, *Response, Advice) error

type Entrypoint func(*Vars, *Agent, *UserInput) error

type Request struct {
	// The name/command of the active agent
	Agent   string
	Command string

	Messages []*Message

	RawInput *UserInput

	//
	ImageQuality string
	ImageSize    string
	ImageStyle   string

	ExtraParams map[string]any

	// ctx is either the client or server context. It should only
	// be modified via copying the whole Request using Clone or WithContext.
	// It is unexported to prevent people from using Context wrong
	// and mutating the contexts held by callers of the same request.
	ctx context.Context
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
// To create a new request with a context, use [NewRequestWithContext].
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
	r2.Messages = r.Messages
	r2.RawInput = r.RawInput

	return r2
}

type ResponseWriter interface {
	AddMessage(*Response, *Agent)
	SetResult(*Result)
}

type Response struct {
	// A list of message objects generated during the conversation
	// with a sender field indicating which Agent the message originated from.
	Messages []*Message

	// Transfer  bool
	// NextAgent string

	// The last agent to handle a message
	Agent *Agent

	Result *Result
}

func (r *Response) LastMessage() *Message {
	if len(r.Messages) > 0 {
		return r.Messages[len(r.Messages)-1]
	}
	return nil
}

type Agentic interface {
	Serve(*Request, *Response) error
}

// Result encapsulates the possible return values for agent/function.
type Result struct {
	// The result value as a string
	Value string

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
	// Root string `json:"root"`

	// web resource base url
	// http://localhost:18080/resource/
	// Bases []string `json:"bases"`

	// Token string `json:"token"`

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
	log.Debugf("Loading agent resource: %s\n", p)
	var ar AgentResource
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, err
	}
	log.Debugf("Agent resource loaded: %+v\n", ar)
	return &ar, nil
}
