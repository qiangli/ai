package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

type LLMAdapter func(context.Context, *Request) (*Response, error)

type AdapterRegistry interface {
	Get(key string) (LLMAdapter, error)
}

// Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// for example, OpenAI: gpt-4.1-mini, gpt-4.1, o3
type Level = api.Level

const (
	// any of L1/L2/L3
	Any Level = "any"

	L1 Level = "L1"
	L2 Level = "L2"
	L3 Level = "L3"

	//
	Image Level = "image"
	TTS   Level = "tts"
)

type Request struct {
	Agent string

	Model *api.Model

	Messages []*api.Message

	MaxTurns int

	RunTool api.ToolRunner

	Tools []*api.ToolFunc

	// Experimenal
	Vars *api.Vars

	Arguments map[string]any

	// get api token for LLM model
	Token func() string

	// openai v3
	Instruction string
	Query       string
}

func (r *Request) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent: %s\n", r.Agent))
	if r.Model != nil {
		sb.WriteString(fmt.Sprintf("Model: %s/%s\n", r.Model.Provider, r.Model.Model))
	}
	sb.WriteString(fmt.Sprintf("MaxTurns: %d\n", r.MaxTurns))
	sb.WriteString(fmt.Sprintf("Tools: %d\n", len(r.Tools)))
	sb.WriteString(fmt.Sprintf("Messages: %d\n", len(r.Messages)))

	return sb.String()
}

type Response struct {
	Role string

	Result *api.Result
}

func (r *Response) String() string {
	var sb strings.Builder
	// sb.WriteString(fmt.Sprintf("ContentType: %s\n", r.ContentType))
	// sb.WriteString(fmt.Sprintf("Content: %d\n", len(r.Content)))
	// sb.WriteString(fmt.Sprintf("Agent: %s\n", r.Agent))
	sb.WriteString(fmt.Sprintf("Role: %s\n", r.Role))
	if r.Result != nil {
		sb.WriteString(fmt.Sprintf("Model: %s\n", r.Result))
	}
	return sb.String()
}
