package swarm

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/api/model"
)

func initModels(app *api.AppConfig, alias string) (func(level string) (*api.Model, error), error) {
	cfg, err := loadModels(app, alias)
	if err != nil {
		return nil, err
	}

	var modelMap = make(map[string]*model.Model)
	for k, v := range cfg.Models {
		modelMap[k] = &model.Model{
			Provider: v.Provider,
			Model:    v.Model,
			BaseUrl:  v.BaseUrl,
			ApiKey:   v.ApiKey,
		}
	}

	// set keys
	provide := func(v *api.Model) (*api.Model, error) {
		m := v.Clone()
		if apiKey, ok := app.ApiKeys[m.Provider]; ok {
			m.ApiKey = apiKey
			return m, nil
		}
		return nil, fmt.Errorf("no api key provided: %s %s", alias, m.Model)
	}

	return func(level string) (*api.Model, error) {
		if v, ok := modelMap[level]; ok {
			return provide(v)
		}
		// special treatment
		if level == model.Any {
			for _, k := range []string{model.L1, model.L2, model.L3} {
				if v, ok := modelMap[k]; ok {
					return provide(v)
				}
			}
		}
		return nil, fmt.Errorf("not found: %s", level)
	}, nil
}

// return early if found
func loadModels(app *api.AppConfig, alias string) (*model.ModelsConfig, error) {
	var modelsCfg = make(map[string]*model.ModelsConfig)

	// built in resource
	if err := LoadResourceModelsConfig(resourceFS, modelsCfg); err != nil {
		log.Debugf("failed to load tools from web resource: %v\n", err)
	} else if cfg, ok := modelsCfg[alias]; ok {
		return cfg, nil
	}

	// external/custom
	if err := LoadFileModelsConfig(app.Base, modelsCfg); err != nil {
		log.Debugf("failed to load custom tools: %v\n", err)
	} else if cfg, ok := modelsCfg[alias]; ok {
		return cfg, nil
	}

	// web
	// https://ai.dhnt.io/models
	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebModelsConfig(app.AgentResource.Resources, modelsCfg); err != nil {
			log.Debugf("failed to load tools from web resource: %v\n", err)
		} else if cfg, ok := modelsCfg[alias]; ok {
			return cfg, nil
		}
	}

	return nil, fmt.Errorf("No LLM configuration found")
}

func ListModels(app *api.AppConfig) (map[string]*model.ModelsConfig, error) {
	var modelsCfg = make(map[string]*model.ModelsConfig)
	if err := LoadResourceModelsConfig(resourceFS, modelsCfg); err != nil {
		log.Debugf("failed to load tools from web resource: %v\n", err)
	}

	if err := LoadFileModelsConfig(app.Base, modelsCfg); err != nil {
		log.Debugf("failed to load custom tools: %v\n", err)
	}

	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebModelsConfig(app.AgentResource.Resources, modelsCfg); err != nil {
			log.Debugf("failed to load tools from web resource: %v\n", err)
		}
	}

	return modelsCfg, nil
}

func LoadResourceModelsConfig(fs embed.FS, aliases map[string]*model.ModelsConfig) error {
	rs := &ResourceStore{
		FS:   fs,
		Base: "resource",
	}
	return LoadModelsAsset(rs, "models", aliases)
}

func LoadFileModelsConfig(base string, aliases map[string]*model.ModelsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	// check if abs exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.Debugf("path does not exist: %s\n", abs)
		return nil
	}

	fs := &FileStore{
		Base: abs,
	}
	return LoadModelsAsset(fs, "models", aliases)
}

func LoadWebModelsConfig(resources []*api.Resource, aliases map[string]*model.ModelsConfig) error {
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

func LoadModelsData(data [][]byte) (*model.ModelsConfig, error) {
	merged := &model.ModelsConfig{}

	for _, v := range data {
		cfg := &model.ModelsConfig{}
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
		if v.ApiKey == "" {
			v.ApiKey = merged.ApiKey
		}
		if v.BaseUrl == "" {
			v.BaseUrl = merged.BaseUrl
		}
		if v.Provider == "" {
			v.Provider = merged.Provider
		}
		if v.Model == "" {
			v.Model = merged.Model
		}
		//
		if len(v.Features) > 0 {
			if _, ok := v.Features[model.Feature(model.OutputTypeImage)]; ok {
				v.Type = model.OutputTypeImage
			}
		}
		// validate
		if v.Provider == "" {
			return nil, fmt.Errorf("missing provider")
		}
	}

	return merged, nil
}

func LoadModelsAsset(as api.AssetStore, base string, m map[string]*model.ModelsConfig) error {
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
			cfg, err := LoadModelsData([][]byte{b})
			if err != nil {
				return fmt.Errorf("failed to load %q: %v", name, err)
			}
			//
			alias := cfg.Alias
			if alias == "" {
				alias = strings.TrimSuffix(name, filepath.Ext(name))
				cfg.Alias = alias
			}
			m[alias] = cfg
		}
	}

	return nil
}
