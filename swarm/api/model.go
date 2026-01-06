package api

import (
	"fmt"
	"strings"
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

// Model set and level
// ^[a-zA-Z0-9_]+$
type Setlevel string

func NewSetlevel(set, level string) Setlevel {
	return Setlevel(set + "/" + level)
}

func (r Setlevel) String() string {
	return string(r)
}

// return default/any for empty string
func (r Setlevel) Decode() (string, string) {
	s := string(r)
	parts := strings.SplitN(s, "/", 2)

	var set = parts[0]
	var level string
	if len(parts) > 1 {
		level = parts[1]
	}
	if level == "" {
		level = "any"
	}
	return set, level
}

func (r Setlevel) Equal(s string) bool {
	s1, l1 := r.Decode()
	s2, l2 := Setlevel(s).Decode()
	if s1 == s2 && l1 == l2 {
		return true
	}
	if s1 == s2 && (l1 == "any" || l2 == "any") {
		return true
	}
	return false
}

var Levels = []Level{L1, L2, L3, Image, TTS}

type Model struct {
	Set   string `json:"set"`
	Level Level  `json:"level"`

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

type ModelConfig struct {
	// LLM model
	Model string `yaml:"model" json:"model"`

	// LLM service provider: openai | gemini | anthropic
	Provider string `yaml:"provider" json:"provider"`
	BaseUrl  string `yaml:"base_url" json:"base_url"`

	// name of api key for looking up api access token
	ApiKey string `yaml:"api_key" json:"api_key"`
}
