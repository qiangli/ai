package swarm

// import (
// 	"context"
// 	"fmt"

// 	"github.com/qiangli/ai/api"
// 	// "github.com/qiangli/ai/internal/swarm/vfs"
// )

// type UserInput = api.UserInput
// type Message = api.Message
// type ToolFunc = api.ToolFunc
// type DBCred = api.DBCred
// type Model = api.Model
// type Result = api.Result

// const (
// 	VarsEnvContainer = "container"
// 	VarsEnvHost      = "host"
// )

// type Vars struct {
// 	OS        string            `json:"os"`
// 	Arch      string            `json:"arch"`
// 	ShellInfo map[string]string `json:"shell_info"`
// 	OSInfo    map[string]string `json:"os_info"`

// 	UserInfo map[string]string `json:"user_info"`

// 	Workspace string `json:"workspace"`
// 	Repo      string `json:"repo"`
// 	Home      string `json:"home"`
// 	Temp      string `json:"temp"`

// 	// WorkDir   string `json:"workdir"`

// 	// EnvType indicates the environment type where the agent is running
// 	// It can be "container" for Docker containers or "host" for the host machine
// 	EnvType string `json:"env_type"`

// 	DBCred *DBCred

// 	Roots []string `json:"roots"`

// 	// per agent
// 	Extra map[string]any `json:"extra"`

// 	Models map[api.Level]*Model `json:"models"`

// 	McpServerUrl string `json:"mcp_server_url"`

// 	//
// 	// FS            vfs.FileSystem
// 	// McpServerTool *McpServerTool

// 	ToolRegistry map[string]*ToolFunc `json:"tool_registry"`
// 	FuncRegistry map[string]Function  `json:"func_registry"`
// }

// func NewVars() *Vars {
// 	return &Vars{
// 		Extra: map[string]any{},
// 	}
// }

// func (r *Vars) Get(key string) any {
// 	if r.Extra == nil {
// 		return nil
// 	}
// 	return r.Extra[key]
// }

// func (r *Vars) GetString(key string) string {
// 	if r.Extra == nil {
// 		return ""
// 	}
// 	v, ok := r.Extra[key]
// 	if !ok {
// 		return ""
// 	}
// 	s, ok := v.(string)
// 	if !ok {
// 		return fmt.Sprintf("%v", v)
// 	}
// 	return s
// }

// // Swarm Agents can call functions directly.
// // Function should usually return a string values.
// // If a function returns an Agent, execution will be transferred to that Agent.
// type Function = func(context.Context, *Vars, string, map[string]any) (*Result, error)

// type Advice func(*Vars, *Request, *Response, Advice) error

// type Entrypoint func(*Vars, *Agent, *UserInput) error

// type Request struct {
// 	// The name/command of the active agent
// 	Agent   string
// 	Command string

// 	Message *Message

// 	RawInput *UserInput

// 	//
// 	ImageQuality string
// 	ImageSize    string
// 	ImageStyle   string

// 	ExtraParams map[string]any

// 	// ctx is either the client or server context. It should only
// 	// be modified via copying the whole Request using Clone or WithContext.
// 	// It is unexported to prevent people from using Context wrong
// 	// and mutating the contexts held by callers of the same request.
// 	ctx context.Context
// }

// // Context returns the request's context. To change the context, use
// // [Request.Clone] or [Request.WithContext].
// //
// // The returned context is always non-nil; it defaults to the
// // background context.
// //
// // For outgoing client requests, the context controls cancellation.
// //
// // For incoming server requests, the context is canceled when the
// // client's connection closes, the request is canceled (with HTTP/2),
// // or when the ServeHTTP method returns.
// func (r *Request) Context() context.Context {
// 	if r.ctx != nil {
// 		return r.ctx
// 	}
// 	return context.Background()
// }

// // WithContext returns a shallow copy of r with its context changed
// // to ctx. The provided ctx must be non-nil.
// //
// // For outgoing client request, the context controls the entire
// // lifetime of a request and its response: obtaining a connection,
// // sending the request, and reading the response headers and body.
// //
// // To create a new request with a context, use [NewRequestWithContext].
// // To make a deep copy of a request with a new context, use [Request.Clone].
// func (r *Request) WithContext(ctx context.Context) *Request {
// 	if ctx == nil {
// 		panic("nil context")
// 	}
// 	r2 := new(Request)
// 	*r2 = *r
// 	r2.ctx = ctx
// 	return r2
// }

// // Clone returns a shallow copy of r with its context changed to ctx.
// // The provided ctx must be non-nil.
// //
// // Clone only makes a shallow copy of the Body field.
// //
// // For an outgoing client request, the context controls the entire
// // lifetime of a request and its response: obtaining a connection,
// // sending the request, and reading the response headers and body.
// func (r *Request) Clone(ctx context.Context) *Request {
// 	if ctx == nil {
// 		panic("nil context")
// 	}
// 	r2 := new(Request)
// 	*r2 = *r
// 	r2.ctx = ctx
// 	r2.Agent = r.Agent
// 	r2.Message = r.Message
// 	r2.RawInput = r.RawInput

// 	return r2
// }

// type Response struct {
// 	// A list of message objects generated during the conversation
// 	// with a sender field indicating which Agent the message originated from.
// 	Messages []*Message

// 	// Transfer  bool
// 	// NextAgent string

// 	// The last agent to handle a message
// 	Agent *Agent

// 	Result *Result
// }

// func (r *Response) LastMessage() *Message {
// 	if len(r.Messages) > 0 {
// 		return r.Messages[len(r.Messages)-1]
// 	}
// 	return nil
// }

// type Agentic interface {
// 	Serve(*Request, *Response) error
// }
