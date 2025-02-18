package swarm

import (
	"context"
	"fmt"

	// "github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
)

// type AgentRegistry map[string]*Agent

// var Agents = AgentRegistry{}

type UserInput = api.Request

type Message = llm.Message

// type Request = llm.Request

type ToolFunc = llm.ToolFunc

// type ToolCall = llm.ToolCall
type ToolCall = llm.ToolCall

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

type DBCred = api.DBCred

type Vars struct {
	OS        string
	Arch      string
	ShellInfo map[string]string
	OSInfo    map[string]string

	UserInfo map[string]string

	Workspace string
	WorkDir   string
	Env       string

	DBCred *DBCred

	// per agent
	// Input  *UserInput
	// Role   string
	// Prompt string
	// Model  *Model
	Extra map[string]any

	Models map[string]*Model

	Functions    map[string]*ToolFunc
	FuncRegistry map[string]Function
}

func NewVars() *Vars {
	return &Vars{
		Extra: map[string]any{},
	}
}

func (r *Vars) Get(key string) any {
	if r.Extra == nil {
		return nil
	}
	return r.Extra[key]
}

func (r *Vars) GetString(key string) string {
	if r.Extra == nil {
		return ""
	}
	v, ok := r.Extra[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

// Swarm Agents can call functions directly.
// Function should usually return a string (values will be attempted to be cast as a string).
// If a function returns an Agent, execution will be transferred to that Agent.
// type Function func(*Input) *Output

// type Function struct {
// 	Name        string
// 	Description string
// 	Parameters  map[string]any
// }

type Function = func(context.Context, *Agent, string, map[string]any) (*Result, error)

// // Agent instructions are directly converted into the system prompt of a conversation.
// // The instructions can either be a regular string, or a function that returns a string.
// type Instruction struct {
// 	Content string
// 	Func    func(Vars) (string, error)
// }

// func (r Instruction) Get(vars Vars) (string, error) {
// 	if r.Func != nil {
// 		return r.Func(vars)
// 	}
// 	return r.Content, nil
// }

type Request struct {
	Agent string

	Message *Message

	RawInput *UserInput

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
	r2.Message = r.Message
	r2.RawInput = r.RawInput

	return r2
}

type Response struct {
	// A list of message objects generated during the conversation
	// with a sender field indicating which Agent the message originated from.
	Messages []*Message

	Transfer  bool
	NextAgent string

	// The last agent to handle a message
	Agent *Agent
}

func (r *Response) LastMessage() *Message {
	if len(r.Messages) > 0 {
		return r.Messages[len(r.Messages)-1]
	}
	return nil
}

// Result encapsulates the possible return values for an agent function.
// type Result struct {
// 	// The result value as a string
// 	Value string

// 	// The current agent instance, if applicable
// 	Agent *Agent

// 	// The agent name to transfer to for StateTransfer
// 	NextAgent string

// 	State State
// }

type State = api.State
type Result = api.Result

// type State int

// const (
// 	StateUnknown State = iota

// 	StateExit
// 	StateTransfer
// 	StateInputWait
// )

type Model = api.Model

type Agentic interface {
	Run(*Request, *Response) error
}

// type Descriptor interface {
// 	Help() string
// 	Describe() string
// 	Version() string
// }

// type Commander interface {
// 	Execute(Text) Text
// }

// type Treelike interface {
// 	Sub() []Agentic
// }

// type ToolDefinition struct {
// 	Name        string         `json:"name"`
// 	Description string         `json:"description"`
// 	Parameters  ToolParameters `json:"parameters"`
// }

// type ToolParameters struct {
// 	Type       string         `json:"type"`
// 	Properties map[string]any `json:"properties"`
// 	Required   []string       `json:"required"`
// }

type Advice func(*Vars, *Request, *Response, Advice) error

type Entrypoint func(*Vars, *Agent, *UserInput) error

type AgentFunc func(string, *Vars) (*Agent, error)

// type ToolConfig = internal.ToolConfig
