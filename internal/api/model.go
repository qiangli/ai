package api

import (
	"github.com/openai/openai-go"
)

type Tool openai.ChatCompletionToolParam

type Tools []openai.ChatCompletionToolParam

type ModelType string

const (
	ModelTypeUnknown ModelType = ""
	ModelTypeText    ModelType = "text"
	ModelTypeImage   ModelType = "image"
)

type Model struct {
	Type    ModelType
	Name    string
	BaseUrl string
	ApiKey  string
}

// Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// for example, OpenAI: gpt-4o-mini, gpt-4o, gpt-o1
type Level int

const (
	L0 Level = iota
	L1
	L2
	L3
)
