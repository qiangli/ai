package api

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
	Config *ModelsConfig
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

type ModelsConfig struct {
	Owner string `yaml:"owner"`

	Alias string `yaml:"alias"`

	// default for Models
	Model    string `yaml:"model"`
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`
	ApiKey   string `yaml:"api_key"`

	Models map[string]*ModelConfig `yaml:"models"`
}

type ModelConfig struct {
	// key of models map in models config
	// Name string `yaml:"name"`
	// Features map[Feature]bool `yaml:"features" json:"features"`

	// //
	// Type        string `yaml:"type"`
	// Type OutputType `yaml:"type" json:"type"`

	// Description string `yaml:"description"`

	// Provider    string `yaml:"provider"`
	// Model       string `yaml:"model"`
	// BaseUrl     string `yaml:"baseUrl"`
	// ApiKey      string `yaml:"apiKey"`

	Provider string `yaml:"provider" json:"provider"`

	Model   string `yaml:"model" json:"model"`
	BaseUrl string `yaml:"base_url" json:"baseUrl"`
	ApiKey  string `yaml:"api_key" json:"apiKey"`
}
