package conf

import (
	"fmt"
	"os"
	"path"

	// "time"

	"dario.cat/mergo"
	// "github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
)

type ModelCacheKey struct {
	Owner string
	// set
	Models string
}

// var (
// 	modelCache = expirable.NewLRU[ModelCacheKey, map[string]*api.Model](10000, nil, time.Second*180)
// )

func LoadModel(owner, set, level string, assets api.AssetManager) (*api.Model, error) {
	provide := func(mc *api.AppConfig, level string) (*api.Model, error) {
		c, ok := mc.Models[level]
		if !ok {
			if level == api.Any {
				for _, k := range []string{api.L1, api.L2, api.L3} {
					if v, ok := mc.Models[k]; ok {
						c = v
						break
					}
				}
			}
		}

		if c == nil {
			return nil, fmt.Errorf("model not found: %s/%s", mc.Set, level)
		}

		m := &api.Model{
			Set:   set,
			Level: level,
			//
			Provider: c.Provider,
			Model:    c.Model,
			BaseUrl:  c.BaseUrl,
			ApiKey:   c.ApiKey,
		}

		return m, nil
	}

	//
	mc, err := assets.FindModels(owner, set)
	if err != nil {
		return nil, err
	}
	if mc != nil {
		return provide(mc, level)
	}

	return nil, fmt.Errorf("model not found: %s/%s", set, level)
}

func loadModelsData(data [][]byte) (*api.AppConfig, error) {
	merged := &api.AppConfig{}

	for _, v := range data {
		cfg := &api.AppConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	// fill defaults
	for _, v := range merged.Models {
		if v.BaseUrl == "" {
			v.BaseUrl = merged.BaseUrl
		}
		if v.Provider == "" {
			v.Provider = merged.Provider
		}
		// if v.Model == "" {
		// 	v.Model = merged.Model
		// }
		if v.ApiKey == "" {
			v.ApiKey = merged.ApiKey
		}

		// validate
		// provider is required for model
		if v.Provider == "" {
			return nil, fmt.Errorf("missing provider: %s", merged.Set)
		}
	}

	return merged, nil
}

func listModelsATM(owner string, as api.ATMSupport, models map[string]*api.AppConfig) error {
	recs, err := as.ListModels(owner)
	if err != nil {
		return err
	}

	// not found
	if len(recs) == 0 {
		return nil
	}

	for _, v := range recs {
		mc, err := loadModelsData([][]byte{[]byte(v.Content)})
		if err != nil {
			return err
		}
		if mc == nil || len(mc.Models) == 0 {
			return fmt.Errorf("invalid config. no model defined: %s", v.Name)
		}
		//
		if mc.Set == "" {
			mc.Set = modelName(v.Name)
		}
		if _, ok := models[mc.Set]; ok {
			continue
		}

		models[mc.Set] = mc
	}
	return nil
}

func listModelsAsset(as api.AssetFS, base string, models map[string]*api.AppConfig) error {
	dirs, err := as.ReadDir(base)
	if err != nil {
		return err
	}

	// not found
	if len(dirs) == 0 {
		return nil
	}

	for _, v := range dirs {
		if v.IsDir() {
			continue
		}
		content, err := as.ReadFile(path.Join(base, v.Name()))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read model asset %s: %w", v.Name(), err)
		}
		if len(content) == 0 {
			// error?
			continue
		}

		mc, err := loadModelsData([][]byte{content})
		if err != nil {
			return err
		}
		if mc == nil || len(mc.Models) == 0 {
			continue
		}

		//
		if mc.Set == "" {
			mc.Set = modelName(v.Name())
		}
		if _, ok := models[mc.Set]; ok {
			continue
		}
		models[mc.Set] = mc
	}
	return nil
}

// default openai/gemini/anthropic models
var DefaultModels = map[string]*api.Model{
	"anthropic": {
		Provider: "anthropic",
		BaseUrl:  "https://api.anthropic.com/",
		ApiKey:   "anthropic",
		Model:    "claude-3-5-haiku-latest",
	},
	"gemini": {
		Provider: "gemini",
		BaseUrl:  "https://generativelanguage.googleapis.com/v1beta/",
		ApiKey:   "gemini",
		Model:    "gemini-2.0-flash-lite",
	},
	"openai": {
		Provider: "openai",
		BaseUrl:  "https://api.openai.com/v1/",
		ApiKey:   "openai",
		Model:    "gpt-5-nano",
	},
}
