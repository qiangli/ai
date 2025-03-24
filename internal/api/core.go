package api

import (
	"context"
	"fmt"
	"strings"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
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

func (r *Result) String() string {
	var sb strings.Builder
	if r.State != StateUnknown {
		sb.WriteString(r.State.String())
	}
	if r.NextAgent != "" {
		sb.WriteString(fmt.Sprintf("%s\n", r.NextAgent))
	}
	if r.Value != "" {
		sb.WriteString(fmt.Sprintf("Value: %s\n", r.Value))
	}
	return strings.TrimSpace(sb.String())
}

type State int

func (s State) String() string {
	switch s {
	case StateUnknown:
		return "DEFAULT"
	case StateExit:
		return "EXIT"
	case StateTransfer:
		return "TRANSFER"
	case StateInputWait:
		return "INPUT_WAIT"
	default:
		return "INVALID"
	}
}

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

	ImageQuality string
	ImageSize    string
	ImageStyle   string

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

// ToolDescriptor is a description of a tool function.
type ToolDescriptor struct {
	Name        string
	Description string
	Parameters  map[string]any

	Body string
}

type ToolFunc struct {
	Type string

	// func class
	// Agent name
	// MCP server name
	// Virtual file system name
	// Container name
	// Virtual machine name
	Kit string

	// func name
	Name        string
	Description string
	Parameters  map[string]any

	Body string
}

// ID returns a unique identifier for the tool function,
// combining the tool name and function name.
func (r *ToolFunc) ID() string {
	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
}

type Response struct {
	ContentType string
	Content     string

	Agent   string
	Display string
	Role    string

	Result *Result
}
