package model

import (
	"maps"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"
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
	Name    string `yaml:"name"`
	BaseUrl string `yaml:"base_url"`
	ApiKey  string `yaml:"api_key"`

	Models map[Level]*Model
}

// Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// for example, OpenAI: gpt-4.1-mini, gpt-4.1, o3
type Level string

const (
	// L0 Level = iota
	L1 Level = "L1"
	L2 Level = "L2"
	L3 Level = "L3"
	//
	Image Level = "Image"
)

var Levels = []Level{L1, L2, L3, Image}

type Model struct {
	Features map[Feature]bool `yaml:"features" json:"features"`

	// output
	Type OutputType `yaml:"type" json:"type"`

	// [provider/]model
	Name string `yaml:"name" json:"name"`

	BaseUrl string `yaml:"base_url" json:"baseUrl"`
	ApiKey  string `yaml:"api_key" json:"apiKey"`
}

func (r *Model) Model() string {
	_, m := r.split()
	return m
}

func (r *Model) Provider() string {
	p, _ := r.split()
	return p
}

func (r *Model) Clone() *Model {
	clone := &Model{
		Features: make(map[Feature]bool, len(r.Features)),
		Type:     r.Type,
		Name:     r.Name,
		BaseUrl:  r.BaseUrl,
		ApiKey:   r.ApiKey,
	}
	maps.Copy(clone.Features, r.Features)
	return clone
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

// LoadModels loads all aliases for models in a given baes directory
func LoadModels(base string) (map[string]*ModelsConfig, error) {
	files, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var m = make(map[string]*ModelsConfig)

	// read all yaml files in the base dir
	for _, v := range files {
		if v.IsDir() {
			continue
		}
		name := v.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			// read the file content
			b, err := os.ReadFile(filepath.Join(base, name))
			if err != nil {
				return nil, err
			}
			cfg, err := loadModelsData([][]byte{b})
			//
			alias := cfg.Name
			if alias == "" {
				alias = strings.TrimSuffix(name, filepath.Ext(name))

			}
			m[alias] = cfg
		}
	}

	return m, nil
}

func loadModelsData(data [][]byte) (*ModelsConfig, error) {
	merged := &ModelsConfig{}

	for _, v := range data {
		cfg := &ModelsConfig{}
		exp := os.ExpandEnv(string(v))
		if err := yaml.Unmarshal([]byte(exp), cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	// fill defaults
	for _, v := range merged.Models {
		v.ApiKey = merged.ApiKey
		v.BaseUrl = merged.BaseUrl
		//
		if len(v.Features) > 0 {
			if _, ok := v.Features[Feature(OutputTypeImage)]; ok {
				v.Type = OutputTypeImage
			}
		}
	}

	return merged, nil
}
