package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

// // Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// // for example, OpenAI: gpt-4.1-mini, gpt-4.1, o3
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

type LLMRequest struct {
	Agent string

	Model *api.Model

	Messages []*api.Message

	MaxTurns int

	RunTool func(ctx context.Context, name string, props map[string]any) (*api.Result, error)

	Tools []*api.ToolFunc

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

	Result *api.Result
}
