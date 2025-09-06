package model

import (
	"maps"
)

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

type ModelsConfig struct {
	Alias string `yaml:"alias" json:"alias"`

	// model
	Model string `yaml:"model" json:"model"`

	// default for Models
	Provider string `yaml:"provider" json:"provider"`
	BaseUrl  string `yaml:"base_url" json:"baseUrl"`
	ApiKey   string `yaml:"api_key" json:"apiKey"`

	Models map[Level]*Model `yaml:"models" json:"models"`
}

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
	Features map[Feature]bool `yaml:"features" json:"features"`

	// output
	Type OutputType `yaml:"type" json:"type"`

	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`

	BaseUrl string `yaml:"base_url" json:"baseUrl"`
	ApiKey  string `yaml:"api_key" json:"apiKey"`
}

func (r *Model) Clone() *Model {
	clone := &Model{
		Features: make(map[Feature]bool, len(r.Features)),
		Type:     r.Type,
		Provider: r.Provider,
		Model:    r.Model,
		BaseUrl:  r.BaseUrl,
		ApiKey:   r.ApiKey,
	}
	maps.Copy(clone.Features, r.Features)
	return clone
}
