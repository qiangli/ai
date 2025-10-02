package conf

import (
	"fmt"
	"path/filepath"
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

func loadModel(auth *api.User, owner, models, model string, secrets api.SecretStore) (*api.Model, error) {
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

	mc, err := retrieveActiveModelsConfig(owner, alias)
	if err != nil {
		return nil, err
	}
	if mc != nil {
		return provide(mc, level)
	}

	// system default
	if v, ok := standardModels[alias]; ok {
		return provide(v, level)
	}

	return nil, fmt.Errorf("model not found: %s %s", alias, level)
}

func loadModelsAsset(as api.AssetStore, base string, m map[string]*api.ModelsConfig) error {
	files, err := as.ReadDir(base)
	if err != nil {
		return err
	}

	// read all yaml files in the base dir
	for _, v := range files {
		if v.IsDir() {
			continue
		}
		name := v.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			// read the file content
			b, err := as.ReadFile(filepath.Join(base, name))
			if err != nil {
				return err
			}
			mc, err := loadModelsData([][]byte{b})
			if err != nil {
				return fmt.Errorf("failed to load %q: %v", name, err)
			}
			//
			alias := mc.Alias
			if alias == "" {
				alias = strings.TrimSuffix(name, filepath.Ext(name))
				mc.Alias = alias
			}
			m[alias] = mc
		}
	}

	return nil
}

// retrieve model alias
// return nil if not found
func retrieveActiveModelsConfig(
	owner string,
	alias string,
) (*api.ModelsConfig, error) {
	// m, found, err := db.GetActiveModelByEmail(owner, alias)
	// if err != nil {
	// 	return nil, err
	// }
	// //
	// if !found {
	// 	return nil, nil
	// }
	// []byte(m.Content)
	var content []byte
	mc, err := loadModelsData([][]byte{content})
	if err != nil {
		return nil, fmt.Errorf("failed to load models: %s. %v", alias, err)
	}
	if mc == nil || len(mc.Models) == 0 {
		return nil, fmt.Errorf("invalid models config: %s", alias)
	}

	//
	mc.Alias = alias

	return mc, nil
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
