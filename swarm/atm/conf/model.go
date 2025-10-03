package conf

import (
	"fmt"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
)

type ModelCacheKey struct {
	Owner string
	// alias
	Models string
}

var (
	modelCache = expirable.NewLRU[ModelCacheKey, map[string]*api.Model](10000, nil, time.Second*180)
)

func loadModel(auth *api.User, owner, models, model string, secrets api.SecretStore, assets api.AssetManager) (*api.Model, error) {
	provide := func(mc *api.ModelsConfig, level string) (*api.Model, error) {
		c, ok := mc.Models[level]
		if !ok {
			if level == llm.Any {
				for _, k := range []string{llm.L1, llm.L2, llm.L3} {
					if v, ok := mc.Models[k]; ok {
						c = v
						break
					}
				}
			}
		}

		if c == nil {
			return nil, fmt.Errorf("model not found: %s/%s", mc.Alias, level)
		}

		ak, err := secrets.Get(owner, c.ApiKey)
		if err != nil {
			return nil, err
		}

		m := &api.Model{
			Provider: c.Provider,
			Model:    c.Model,
			BaseUrl:  c.BaseUrl,
			ApiKey:   ak,
			Config:   mc,
		}

		return m, nil
	}

	// models takes precedence over model
	split := func() (string, string) {
		// models: alias[/level]
		alias, level := split2(models, "/", "")

		// model: [alias/]level
		parts := strings.SplitN(model, "/", 2)
		if len(parts) == 2 {
			return nvl(alias, parts[0]), nvl(level, parts[1])
		}
		return nvl(alias, "default"), nvl(level, model)
	}

	// alias/level
	alias, level := split()

	mc, err := assets.FindModels(owner, alias)
	if err != nil {
		return nil, err
	}
	if mc != nil {
		return provide(mc, level)
	}

	return nil, fmt.Errorf("model not found: %s %s", alias, level)
}

func loadModelsData(data [][]byte) (*api.ModelsConfig, error) {
	merged := &api.ModelsConfig{}

	for _, v := range data {
		cfg := &api.ModelsConfig{}
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
		if v.Model == "" {
			v.Model = merged.Model
		}
		if v.ApiKey == "" {
			v.ApiKey = merged.ApiKey
		}

		// validate
		// provider is required for model
		if v.Provider == "" {
			return nil, fmt.Errorf("missing provider: %s", merged.Alias)
		}
	}

	return merged, nil
}
