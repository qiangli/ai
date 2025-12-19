package api

import (
	"context"
	"time"
)

// Action handler
type Handler interface {
	Serve(*Request, *Response) error
}

type HandlerFunc func(req *Request, res *Response) error

func (f HandlerFunc) Serve(req *Request, res *Response) error {
	return f(req, res)
}

type Middleware func(*Agent, Handler) Handler

type Message struct {
	// message id
	ID string `json:"id"`

	// thread id
	Session string    `json:"session"`
	Created time.Time `json:"created"`

	// data
	ContentType string `json:"content_type"`
	Content     string `json:"content"`

	// system | assistant | user
	Role string `json:"role"`

	// user/agent
	Sender string `json:"sender"`
}
type Request struct {
	// parent agent
	Agent *Agent `json:"agent"`

	// active action name
	Name      string     `json:"name"`
	Arguments *Arguments `json:"arguments"`

	// LLM
	Model *Model `json:"model"`

	Messages []*Message `json:"messages"`

	Runner ActionRunner `json:"-"`

	Tools []*ToolFunc `json:"tools"`

	// get api token for LLM model
	Token func() string `json:"-"`

	// ctx should only be modified via copying the whole request WithContext.
	// It is unexported to prevent people from using Context wrong
	// and mutating the contexts held by callers of the same request.
	ctx context.Context `json:"-"`

	//
	Prompt  string     `json:"prompt"`
	Query   string     `json:"query"`
	History []*Message `json:"history"`
}

func (r *Request) Message() string {
	if r.Arguments == nil {
		return ""
	}
	return r.Arguments.Message()
}

func (r *Request) MaxTurns() int {
	if r.Arguments == nil {
		return 0
	}
	return r.Arguments.GetInt("max_turns")
}

func (r *Request) MemOption() *MemOption {
	var o MemOption
	if r.Arguments == nil {
		return &o
	}
	o.MaxHistory = r.Arguments.GetInt("max_history")
	o.MaxSpan = r.Arguments.GetInt("max_span")
	return &o
}

func NewRequest(ctx context.Context, name string, args map[string]any) *Request {
	req := &Request{
		ctx:       ctx,
		Name:      name,
		Arguments: NewArguments(),
	}
	req.Arguments.AddArgs(args)
	return req
}

// Context returns the request's context.
// To change the context, use [Request.WithContext].
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

// Clone returns a shallow copy of r while ensuring proper copying of slices and maps
func (r *Request) Clone() *Request {
	r2 := new(Request)
	*r2 = *r

	if r.Arguments != nil {
		r2.Arguments = r.Arguments.Clone()
	}
	return r2
}

type Response struct {
	// A list of message objects generated during the conversation
	// with a sender field indicating which Agent the message originated from.
	Messages []*Message `json:"messages"`

	// The last agent to handle a message
	Agent *Agent `json:"agent"`

	Result *Result `json:"result"`
}

// Result encapsulates the possible return values for agent/function.
type Result struct {
	// author of the reponse: assistant
	Role string `json:"role"`
	// The result value as a string
	Value string `json:"value"`

	// media content
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/MIME_types
	MimeType string `json:"mime_type"`
	Content  []byte `json:"content"`

	// transition
	// The agent state
	State State `json:"state"`
	// The agent name to transfer to for StateTransfer
	NextAgent string `json:"next_agent"`

	//
	Actions []*ToolCall `json:"actions"`
}

func (r *Result) String() string {
	return r.Value
}

// // abbreviate trims the string, keeping the beginning and end if exceeding maxLen.
// // after replacing newlines with .
// func abbreviate(s string, maxLen int) string {
// 	s = strings.ReplaceAll(s, "\n", "•")
// 	s = strings.Join(strings.Fields(s), " ")
// 	s = strings.TrimSpace(s)

// 	if len(s) > maxLen {
// 		// Calculate the length for each part
// 		keepLen := (maxLen - 3) / 2
// 		start := s[:keepLen]
// 		end := s[len(s)-keepLen:]
// 		return start + "..." + end
// 	}

// 	return s
// }
