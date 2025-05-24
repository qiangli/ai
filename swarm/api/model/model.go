package model

import (
	"strings"
)

type InputType string
type OutputType string
type Feature string

const (
	//
	InputTypeUnknown InputType = ""
	InputTypeText    InputType = "text"
	InputTypeImage   InputType = "image"

	// feature: bool
	// vision
	// natural language
	// coding
	// "input_text"
	// "input_image"
	// audio/video
	// output_text
	// output_image
	// "caching"
	// "tool_calling"
	// "reasoning"
	// "level1"
	// "leval2"
	// "level3"
	// Cost-optimized
	// realtime
	// Text-to-speech
	// Transcription
	// embeddings

)

type ModelsConfig struct {
	Models map[string]*Model
}

type Model struct {
	Features []Feature

	// [provider/]model
	Name string

	BaseUrl string
	ApiKey  string
}

func (r *Model) Model() string {
	_, m := r.split()
	return m
}

func (r *Model) Provider() string {
	p, _ := r.split()
	return p
}

// [<provider>/]<model>
// openai/gpt-4.1-mini
// gemini/gemini-2.0-flash
func (r *Model) split() (string, string) {
	parts := strings.SplitN(r.Name, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", r.Name
}
