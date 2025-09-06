package llm

import (
	"context"
	"fmt"
	// "maps"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
)

// type Model = model.Model

type InputType string
type OutputType string
type Feature string

const (
	//
	// InputTypeUnknown InputType = ""
	// InputTypeText    InputType = "text"
	// InputTypeImage   InputType = "image"

	//
	OutputTypeUnknown OutputType = ""
	OutputTypeText    OutputType = "text"
	OutputTypeImage   OutputType = "image"

	// feature:
	// vision
	// natural language
	// coding
	// input_text
	// input_image
	// audio/video
	// output_text
	// output_image
	// caching
	// tool_calling
	// reasoning
	// level1
	// leval2
	// level3
	// cost-optimized
	// realtime
	// Text-to-speech
	// Transcription
	// embeddings
)

// Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// for example, OpenAI: gpt-4.1-mini, gpt-4.1, o3
type Level = string

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

var Levels = []Level{L1, L2, L3, Image, TTS}

type Model struct {
	Provider string

	Model   string
	BaseUrl string
	ApiKey  string

	// output
	Type OutputType

	// level name
	Name   string
	Config *api.ModelsConfig
}

func (r *Model) Clone() *Model {
	clone := &Model{
		// Features: make(map[Feature]bool, len(r.Features)),
		Type:     r.Type,
		Provider: r.Provider,
		Model:    r.Model,
		BaseUrl:  r.BaseUrl,
		ApiKey:   r.ApiKey,
	}
	// maps.Copy(clone.Features, r.Features)
	return clone
}

// type LLMConfig struct {
// 	Provider string
// 	// Name     string
// 	BaseUrl string
// 	ApiKey  string

// 	// model aliases
// 	Models map[Level]*Model
// }

// func (config *LLMConfig) Clone() *LLMConfig {
// 	modelsCopy := make(map[Level]*Model, len(config.Models))
// 	for k, v := range config.Models {
// 		modelsCopy[k] = v // shallow copy of the values
// 	}

// 	return &LLMConfig{
// 		Provider: config.Provider,
// 		// Name:     config.Name,
// 		BaseUrl: config.BaseUrl,
// 		ApiKey:  config.ApiKey,
// 		//
// 		Models: modelsCopy,
// 	}
// }

type LLMRequest struct {
	Agent string

	Model *Model

	Messages []*Message

	// // TODO extras: name:value
	// ImageQuality string
	// ImageSize    string
	// ImageStyle   string

	MaxTurns int

	RunTool func(ctx context.Context, name string, props map[string]any) (*Result, error)

	Tools []*ToolFunc

	// Experimenal
	Vars *api.Vars
}

func (r *LLMRequest) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent: %s\n", r.Agent))
	if r.Model != nil {
		sb.WriteString(fmt.Sprintf("Model: %s\n", r.Model.Model))
		sb.WriteString(fmt.Sprintf("BaseUrl: %s\n", r.Model.BaseUrl))
		sb.WriteString(fmt.Sprintf("ApiKey set: %v\n", r.Model.ApiKey != ""))
		sb.WriteString(fmt.Sprintf("Type: %s\n", r.Model.Type))
		// if r.Model.Type == model.OutputTypeImage {
		// 	sb.WriteString(fmt.Sprintf("ImageQuality: %s\n", r.ImageQuality))
		// 	sb.WriteString(fmt.Sprintf("ImageSize: %s\n", r.ImageSize))
		// 	sb.WriteString(fmt.Sprintf("ImageStyle: %s\n", r.ImageStyle))
		// }
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

type LLMResponse struct {
	ContentType string
	Content     string

	Agent   string
	Display string
	Role    string

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

	// // model alias
	// Models string `json:"models"`
}

type ToolFunc struct {
	Kit string

	Type string

	// func name
	Name        string
	Description string
	Parameters  map[string]any

	Body string

	//
	State api.State

	//
	Config *api.ToolsConfig
}

// ID returns a unique identifier for the tool function,
// combining the tool kit and function name.
func (r *ToolFunc) ID() string {
	return fmt.Sprintf("%s__%s", r.Kit, r.Name)
}

// Result encapsulates the possible return values for agent/function.
type Result struct {
	// The result value as a string
	Value string

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/MIME_types
	MimeType string
	Message  string

	// The agent state
	State api.State

	// The agent name to transfer to for StateTransfer
	NextAgent string
}

func (r *Result) String() string {
	var sb strings.Builder
	if r.State != api.StateDefault {
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

type ToolSystem interface {
	Call(context.Context, *api.Vars, *ToolFunc, map[string]any) (*Result, error)
}

// TODO
type MemOption struct {
	MaxHistory int
	MaxSpan    int
}

type MemStore interface {
	Save([]*Message) error
	Load(*MemOption) error
}
