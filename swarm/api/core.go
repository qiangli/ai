package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api/model"
)

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

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

type LLMRequest struct {
	Agent string

	Model *model.Model

	Messages []*Message

	ImageQuality string
	ImageSize    string
	ImageStyle   string

	MaxTurns int
	RunTool  func(ctx context.Context, name string, props map[string]any) (*Result, error)

	Tools []*ToolFunc
}

func (r *LLMRequest) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent: %s\n", r.Agent))
	if r.Model != nil {
		sb.WriteString(fmt.Sprintf("Model: %s\n", r.Model.Name))
		sb.WriteString(fmt.Sprintf("BaseUrl: %s\n", r.Model.BaseUrl))
		sb.WriteString(fmt.Sprintf("ApiKey set: %v\n", r.Model.ApiKey != ""))
		sb.WriteString(fmt.Sprintf("ModelType: %s\n", r.Model.Type))
		if r.Model.Type == model.OutputTypeImage {
			sb.WriteString(fmt.Sprintf("ImageQuality: %s\n", r.ImageQuality))
			sb.WriteString(fmt.Sprintf("ImageSize: %s\n", r.ImageSize))
			sb.WriteString(fmt.Sprintf("ImageStyle: %s\n", r.ImageStyle))
		}
	}
	sb.WriteString(fmt.Sprintf("MaxTurns: %d\n", r.MaxTurns))
	sb.WriteString(fmt.Sprintf("RunTool set: %v\n", r.RunTool != nil))
	sb.WriteString(fmt.Sprintf("Tools count: %d\n", len(r.Tools)))

	sb.WriteString(fmt.Sprintf("Messages count: %d\n", len(r.Messages)))
	// for _, m := range r.Messages {
	// 	sb.WriteString(clipText(m.Content, 80))
	// }
	return sb.String()
}

type Message struct {
	ContentType string
	Content     string

	Role   string
	Sender string
}

type LLMResponse struct {
	ContentType string
	Content     string

	Agent   string
	Display string
	Role    string

	Result *Result
}
