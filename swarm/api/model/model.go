package model

import (
	// "fmt"
	"maps"
	// "os"
	// "path/filepath"
	// "strings"
	// "dario.cat/mergo"
	// "gopkg.in/yaml.v3"
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

// // LoadModels loads all aliases for models in a given baes directory
// func LoadModels(base string) (map[string]*ModelsConfig, error) {
// 	files, err := os.ReadDir(base)
// 	if err != nil {
// 		if os.IsNotExist(err) {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}

// 	var m = make(map[string]*ModelsConfig)

// 	// read all yaml files in the base dir
// 	for _, v := range files {
// 		if v.IsDir() {
// 			continue
// 		}
// 		name := v.Name()
// 		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
// 			// read the file content
// 			b, err := os.ReadFile(filepath.Join(base, name))
// 			if err != nil {
// 				return nil, err
// 			}
// 			cfg, err := LoadModelsData([][]byte{b})
// 			if err != nil {
// 				return nil, fmt.Errorf("failed to load %q: %v", name, err)
// 			}
// 			//
// 			alias := cfg.Alias
// 			if alias == "" {
// 				alias = strings.TrimSuffix(name, filepath.Ext(name))
// 				cfg.Alias = alias
// 			}
// 			m[alias] = cfg
// 		}
// 	}

// 	return m, nil
// }

// func LoadModelsData(data [][]byte) (*ModelsConfig, error) {
// 	merged := &ModelsConfig{}

// 	for _, v := range data {
// 		cfg := &ModelsConfig{}
// 		exp := os.ExpandEnv(string(v))
// 		if err := yaml.Unmarshal([]byte(exp), cfg); err != nil {
// 			return nil, err
// 		}

// 		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
// 			return nil, err
// 		}
// 	}

// 	// fill defaults
// 	for _, v := range merged.Models {
// 		if v.ApiKey == "" {
// 			v.ApiKey = merged.ApiKey
// 		}
// 		if v.BaseUrl == "" {
// 			v.BaseUrl = merged.BaseUrl
// 		}
// 		if v.Provider == "" {
// 			v.Provider = merged.Provider
// 		}
// 		if v.Model == "" {
// 			v.Model = merged.Model
// 		}
// 		//
// 		if len(v.Features) > 0 {
// 			if _, ok := v.Features[Feature(OutputTypeImage)]; ok {
// 				v.Type = OutputTypeImage
// 			}
// 		}
// 		// validate
// 		if v.Provider == "" {
// 			return nil, fmt.Errorf("missing provider")
// 		}
// 	}

// 	return merged, nil
// }
