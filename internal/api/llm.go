package api

import (
	"context"
)

// Result encapsulates the possible return values for an agent function.
type Result struct {
	// The result value as a string
	Value string

	// The current agent instance
	State State

	// The agent name to transfer to for StateTransfer
	NextAgent string
}

type State int

const (
	StateUnknown State = iota

	StateExit
	StateTransfer
	StateInputWait
)

type Request struct {
	Agent string

	ModelType ModelType
	BaseUrl   string
	ApiKey    string
	Model     string

	// History  []*Message
	Messages []*Message

	MaxTurns int
	RunTool  func(ctx context.Context, name string, props map[string]any) (*Result, error)

	Tools []*ToolFunc
}

type Message struct {
	ContentType string
	Content     string

	Role   string
	Sender string
}

type ToolFunc struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type Response struct {
	ContentType string
	Content     string

	Agent   string
	Display string
	Role    string

	Result *Result
}
