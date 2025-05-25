package api

import (
	"github.com/qiangli/ai/swarm/api/model"
)

type LLMConfig struct {
	Model   string
	BaseUrl string
	ApiKey  string

	// model aliases
	Models map[model.Level]*model.Model

	// L1 *model.Model
	// L2 *model.Model
	// L3 *model.Model

	// Image *model.Model
}

// func (cfg *LLMConfig) Clone() *LLMConfig {
// 	n := &LLMConfig{
// 		ApiKey:  cfg.ApiKey,
// 		BaseUrl: cfg.BaseUrl,
// 		Model:   cfg.Model,
// 	}
// 	return n
// }

// func Level1(cfg *LLMConfig) *model.Model {
// 	return CreateModel(cfg, model.L1)
// }

// func Level2(cfg *LLMConfig) *model.Model {
// 	return CreateModel(cfg, model.L2)
// }

// func Level3(cfg *LLMConfig) *model.Model {
// 	return CreateModel(cfg, model.L3)
// }

// func ImageModel(cfg *LLMConfig) *model.Model {
// 	return cfg.Image
// 	// if cfg == nil {
// 	// 	return &model.Model{}
// 	// }
// 	// model := &model.Model{
// 	// 	// Type:    ModelTypeImage,
// 	// 	Name:    cfg.ImageModel,
// 	// 	BaseUrl: cfg.BaseUrl,
// 	// 	ApiKey:  cfg.ApiKey,
// 	// }
// 	// if cfg.ImageApiKey != "" {
// 	// 	model.ApiKey = cfg.ImageApiKey
// 	// }
// 	// if cfg.ImageBaseUrl != "" {
// 	// 	model.BaseUrl = cfg.ImageBaseUrl
// 	// }

// 	// return model
// }

// // CreateModel creates a model with the given configuration and optional level
// func CreateModel(cfg *LLMConfig, opt ...model.Level) *model.Model {
// 	if cfg == nil {
// 		return &model.Model{}
// 	}
// 	m := &model.Model{
// 		// Type:    ModelTypeText,
// 		Name:    cfg.Model,
// 		BaseUrl: cfg.BaseUrl,
// 		ApiKey:  cfg.ApiKey,
// 	}

// 	// default level
// 	level := model.L0
// 	if len(opt) > 0 {
// 		level = opt[0]
// 	}

// 	switch level {
// 	case model.L0:
// 		return m
// 	case model.L1:
// 		if cfg.L1ApiKey != "" {
// 			m.ApiKey = cfg.L1ApiKey
// 		}
// 		if cfg.L1Model != "" {
// 			m.Name = cfg.L1Model
// 		}
// 		if cfg.L1BaseUrl != "" {
// 			m.BaseUrl = cfg.L1BaseUrl
// 		}
// 	case model.L2:
// 		if cfg.L2ApiKey != "" {
// 			m.ApiKey = cfg.L2ApiKey
// 		}
// 		if cfg.L2Model != "" {
// 			m.Name = cfg.L2Model
// 		}
// 		if cfg.L2BaseUrl != "" {
// 			m.BaseUrl = cfg.L2BaseUrl
// 		}
// 	case model.L3:
// 		if cfg.L3ApiKey != "" {
// 			m.ApiKey = cfg.L3ApiKey
// 		}
// 		if cfg.L3Model != "" {
// 			m.Name = cfg.L3Model
// 		}
// 		if cfg.L3BaseUrl != "" {
// 			m.BaseUrl = cfg.L3BaseUrl
// 		}
// 	}

// 	return m
// }
