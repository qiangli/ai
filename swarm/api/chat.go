package api

import (
	"context"
	"fmt"
	"maps"
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
	// parent agent
	Parent *Agent

	// The name of the active agent/tool
	Name string

	// default values
	// agent:tool call args
	Arguments map[string]any

	// Messages []*Message

	RawInput *UserInput

	// ctx should only be modified via copying the whole request WithContext.
	// It is unexported to prevent people from using Context wrong
	// and mutating the contexts held by callers of the same request.
	ctx context.Context
}

func NewRequest(ctx context.Context, name string, input *UserInput) *Request {
	return &Request{
		ctx:      ctx,
		Name:     name,
		RawInput: input,
	}
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

	// // fields
	// if r.Messages != nil {
	// 	r2.Messages = make([]*Message, len(r.Messages))
	// 	copy(r2.Messages, r.Messages)
	// }

	if r.RawInput != nil {
		r2.RawInput = r.RawInput.Clone()
	}

	if r.Arguments != nil {
		r2.Arguments = make(map[string]any, len(r.Arguments))
		maps.Copy(r2.Arguments, r.Arguments)
	}
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

// Result encapsulates the possible return values for agent/function.
type Result struct {
	// The result value as a string
	Value string

	// media content
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/MIME_types
	MimeType string
	Content  []byte

	// transition
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
		sb.WriteString(fmt.Sprintf(" %s", r.NextAgent))
	}
	if r.Value != "" {
		sb.WriteString(fmt.Sprintf(" %s [%v]", abbreviate(r.Value, 64), len(r.Value)))
	}
	if r.MimeType != "" {
		sb.WriteString(fmt.Sprintf(" %s", r.MimeType))
	}
	if len(r.Content) > 0 {
		sb.WriteString(fmt.Sprintf(" %s [%v]", abbreviate(string(r.Content), 64), len(r.Content)))
	}
	s := strings.TrimSpace(sb.String())
	if len(s) == 0 {
		return "<empty>"
	}
	return s
}

// abbreviate trims the string, keeping the beginning and end if exceeding maxLen.
// after replacing newlines with .
func abbreviate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "•")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)

	if len(s) > maxLen {
		// Calculate the length for each part
		keepLen := (maxLen - 3) / 2
		start := s[:keepLen]
		end := s[len(s)-keepLen:]
		return start + "..." + end
	}

	return s
}
