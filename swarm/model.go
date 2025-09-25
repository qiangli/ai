package swarm

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

func initModels(ctx context.Context, app *api.AppConfig) (func(level string) (*api.Model, error), error) {
	var alias = app.Models

	cfg, err := loadModels(ctx, app, alias)
	if err != nil {
		return nil, err
	}

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

		ak := getApiKey(c.ApiKey)

		return &api.Model{
			Provider: c.Provider,
			Model:    c.Model,
			BaseUrl:  c.BaseUrl,
			ApiKey:   ak,
			Config:   mc,
		}, nil
	}

	// model/level
	split := func(s string) (string, string) {
		parts := strings.SplitN(s, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
		return alias, s
	}

	return func(level string) (*api.Model, error) {
		// alias/level
		if strings.Contains(level, "/") {
			alias, level = split(level)
			mc, err := loadModels(ctx, app, alias)
			if err != nil {
				return nil, err
			}
			return provide(mc, level)
		}

		return provide(cfg, level)
	}, nil
}

// return early if found
func loadModels(ctx context.Context, app *api.AppConfig, alias string) (*api.ModelsConfig, error) {
	var modelsCfg = make(map[string]*api.ModelsConfig)

	// web
	// https://ai.dhnt.io/models
	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebModelsConfig(ctx, app.AgentResource.Resources, modelsCfg); err != nil {
			log.GetLogger(ctx).Debugf("failed to load models from web resource: %v\n", err)
		} else if cfg, ok := modelsCfg[alias]; ok {
			return cfg, nil
		}
	}

	// external/custom
	if err := LoadFileModelsConfig(ctx, app.Base, modelsCfg); err != nil {
		log.GetLogger(ctx).Debugf("failed to load custom models: %v\n", err)
	} else if cfg, ok := modelsCfg[alias]; ok {
		return cfg, nil
	}

	// built in resource
	if err := LoadResourceModelsConfig(resourceFS, modelsCfg); err != nil {
		log.GetLogger(ctx).Debugf("failed to load models from web resource: %v\n", err)
	} else if cfg, ok := modelsCfg[alias]; ok {
		return cfg, nil
	}

	return nil, fmt.Errorf("No LLM configuration found")
}

func ListModels(ctx context.Context, app *api.AppConfig) (map[string]*api.ModelsConfig, error) {
	var modelsCfg = make(map[string]*api.ModelsConfig)

	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebModelsConfig(ctx, app.AgentResource.Resources, modelsCfg); err != nil {
			log.GetLogger(ctx).Debugf("failed to load models from web resource: %v\n", err)
		}
	}

	if err := LoadFileModelsConfig(ctx, app.Base, modelsCfg); err != nil {
		log.GetLogger(ctx).Debugf("failed to load custom models: %v\n", err)
	}

	if err := LoadResourceModelsConfig(resourceFS, modelsCfg); err != nil {
		log.GetLogger(ctx).Debugf("failed to load models from local resource: %v\n", err)
	}

	return modelsCfg, nil
}

func LoadResourceModelsConfig(fs embed.FS, aliases map[string]*api.ModelsConfig) error {
	rs := &ResourceStore{
		FS:   fs,
		Base: "resource",
	}
	return LoadModelsAsset(rs, "models", aliases)
}

func LoadFileModelsConfig(ctx context.Context, base string, aliases map[string]*api.ModelsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	// check if abs exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.GetLogger(ctx).Debugf("path does not exist: %s\n", abs)
		return nil
	}

	fs := &FileStore{
		Base: abs,
	}
	return LoadModelsAsset(fs, "models", aliases)
}

func LoadWebModelsConfig(ctx context.Context, resources []*api.Resource, aliases map[string]*api.ModelsConfig) error {
	for _, v := range resources {
		ws := &WebStore{
			Base:  v.Base,
			Token: v.Token,
		}
		if err := LoadModelsAsset(ws, "models", aliases); err != nil {
			return fmt.Errorf("failed to load config. base: %s error: %v\n", v.Base, err)
		}
	}
	return nil
}

func LoadModelsData(data [][]byte) (*api.ModelsConfig, error) {
	merged := &api.ModelsConfig{}

	for _, v := range data {
		cfg := &api.ModelsConfig{}
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
		if v.Model == "" {
			v.Model = merged.Model
		}
		if v.BaseUrl == "" {
			v.BaseUrl = merged.BaseUrl
		}
		if v.Provider == "" {
			v.Provider = merged.Provider
		}
		if v.ApiKey == "" {
			v.ApiKey = merged.ApiKey
		}

		//
		// if len(v.Features) > 0 {
		// 	if _, ok := v.Features[model.Feature(model.OutputTypeImage)]; ok {
		// 		v.Type = model.OutputTypeImage
		// 	}
		// }

		// validate
		if v.Provider == "" {
			return nil, fmt.Errorf("missing provider")
		}
	}
	return merged, nil
}

func LoadModelsAsset(as api.AssetStore, base string, m map[string]*api.ModelsConfig) error {
	files, err := as.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// read all yaml files in the base dir
	for _, v := range files {
		if v.IsDir() {
			continue
		}
		name := v.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			// read the asset content
			b, err := as.ReadFile(path.Join(base, name))
			if err != nil {
				return err
			}
			cfg, err := LoadModelsData([][]byte{b})
			if err != nil {
				return fmt.Errorf("failed to load models %q: %v", name, err)
			}

			//
			if cfg.Alias == "" {
				cfg.Alias = trimYaml(name)
			}
			m[cfg.Alias] = cfg
		}
	}

	return nil
}
