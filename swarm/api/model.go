package api

import (
	"fmt"
)

// Level represents the "intelligence" level of the model.
// i.e. basic, regular, advanced
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

	// api token lookup key
	ApiKey string `json:"api_key"`
}

func (r *Model) String() string {
	return fmt.Sprintf("%s/%s", r.Provider, r.Model)
}

// type ModelsConfig struct {
// 	// model set name
// 	Set string `yaml:"set"`

// 	// provider
// 	Provider string `yaml:"provider"`
// 	BaseUrl  string `yaml:"base_url"`
// 	// name of api lookup key - never the actual api token
// 	ApiKey string `yaml:"api_key"`

// 	Models map[string]*ModelConfig `yaml:"models"`
// }

type ModelsConfig AppConfig

type ModelConfig struct {
	// LLM model
	Model string `yaml:"model"`

	// LLM service provider: openai | gemini | anthropic
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	// name of api key
	ApiKey string `yaml:"api_key"`
}
