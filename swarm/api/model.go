package api

import (
	"fmt"

	"github.com/charmbracelet/x/exp/slice"
)

type InputType string
type OutputType string
type Feature string

type ModelAlias = map[string]*Model
type ModelAliasConfig = map[string]*ModelConfig

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
	// model @agent or resolved provider model name
	// example:
	//   @model
	//   gemini-2.0-flash-lite
	Model string `json:"model"`

	Provider string `json:"provider"`
	BaseUrl  string `json:"base_url"`

	// secret key or LLM provider api key/token if resolved
	ApiKey string `json:"api_key"`
}

func (r *Model) String() string {
	return fmt.Sprintf("%s/%s", r.Provider, r.Model)
}

// TODO feature flags
func (r *Model) IsImage() bool {
	list := []string{"dall-e-2", "dall-e-3", "gpt-image-1"}
	return slice.ContainsAny(list, r.Model)
}

type ModelsConfig struct {
	Alias string `yaml:"alias"`

	// default LLM model for ModelConfig
	Model string `yaml:"model"`

	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`

	Models ModelAliasConfig `yaml:"models"`
}

type ModelConfig struct {
	// LLM model
	Model string `yaml:"model"`

	// LLM service provider: openai | gemini | anthropic
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`
}
