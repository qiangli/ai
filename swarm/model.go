package swarm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/swarm/api/model"
)

func initModels(base, alias string) (func(level string) (*model.Model, error), error) {
	path := filepath.Join(base, "models")
	modelCfg, err := model.LoadModels(path)
	if err != nil {
		return nil, err
	}

	if alias == "" {
		alias = "openai"
	}

	cfg, ok := modelCfg[alias]
	if !ok {
		return nil, fmt.Errorf("models not found for alias %q", alias)
	}

	var cache = make(map[string]*model.Model)
	for k, v := range cfg.Models {
		cache[k] = &model.Model{
			Provider: v.Provider,
			Model:    v.Model,
			BaseUrl:  v.BaseUrl,
			ApiKey:   v.ApiKey,
		}
	}

	// if no models, setup defaults
	if len(cache) == 0 {
		// all levels share same config
		var m model.Model
		switch {
		case os.Getenv("OPENAI_API_KEY") != "":
			m = model.Model{
				Model:    "gpt-5-nano",
				Provider: "openai",
				BaseUrl:  "https://api.openai.com/v1/",
				ApiKey:   os.Getenv("OPENAI_API_KEY"),
			}
		case os.Getenv("GEMINI_API_KEY") != "":
			m = model.Model{
				Model:    "gemini-2.0-flash-lite",
				Provider: "gemini",
				BaseUrl:  "",
				ApiKey:   os.Getenv("GEMINI_API_KEY"),
			}
		case os.Getenv("ANTHROPIC_API_KEY") != "":
			m = model.Model{
				Model:    "claude-3-5-haiku-latest",
				Provider: "anthropic",
				BaseUrl:  "",
				ApiKey:   os.Getenv("ANTHROPIC_API_KEY"),
			}
		default:
		}
		cache[m.Provider] = &m
	}

	if len(cache) == 0 {
		return nil, fmt.Errorf("No LLM configuration found")
	}

	return func(level string) (*model.Model, error) {
		if v, ok := cache[level]; ok {
			return v, nil
		}
		// special treatment
		if level == model.Any {
			for _, k := range []string{model.L1, model.L2, model.L3} {
				if v, ok := cache[k]; ok {
					return v, nil
				}
			}
		}
		return nil, fmt.Errorf("not found: %s", level)
	}, nil
}
