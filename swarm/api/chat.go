package api

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Chat struct {
	// uuid
	ID      string    `json:"id"`
	UserID  string    `json:"userId"`
	Created time.Time `json:"created"`

	// data
	Title string `json:"title"`
}

type Request struct {
	// The name/command of the active agent
	Agent string
	// Command string

	Messages []*Message

	RawInput *UserInput

	Arguments map[string]any

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
	// Message  string

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
