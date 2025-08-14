package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api/model"
)

type LLMConfig struct {
	Provider string
	// Name     string
	BaseUrl string
	ApiKey  string

	// model aliases
	Models map[model.Level]*model.Model
}

func (config *LLMConfig) Clone() *LLMConfig {
	modelsCopy := make(map[model.Level]*model.Model, len(config.Models))
	for k, v := range config.Models {
		modelsCopy[k] = v // shallow copy of the values
	}

	return &LLMConfig{
		Provider: config.Provider,
		// Name:     config.Name,
		BaseUrl: config.BaseUrl,
		ApiKey:  config.ApiKey,
		//
		Models: modelsCopy,
	}
}

type LLMRequest struct {
	Agent string

	Model *model.Model

	Messages []*Message

	// TODO extras: name:value
	ImageQuality string
	ImageSize    string
	ImageStyle   string

	MaxTurns int
	RunTool  func(ctx context.Context, name string, props map[string]any) (*Result, error)

	Tools []*ToolFunc

	// Experimenal
	Vars *Vars
}

func (r *LLMRequest) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Agent: %s\n", r.Agent))
	if r.Model != nil {
		sb.WriteString(fmt.Sprintf("Model: %s\n", r.Model.Model))
		sb.WriteString(fmt.Sprintf("BaseUrl: %s\n", r.Model.BaseUrl))
		sb.WriteString(fmt.Sprintf("ApiKey set: %v\n", r.Model.ApiKey != ""))
		sb.WriteString(fmt.Sprintf("Type: %s\n", r.Model.Type))
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

type LLMResponse struct {
	ContentType string
	Content     string

	Agent   string
	Display string
	Role    string

	Result *Result
}
