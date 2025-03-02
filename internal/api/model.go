package api

type ModelType string

const (
	ModelTypeUnknown ModelType = ""
	ModelTypeText    ModelType = "text"
	ModelTypeImage   ModelType = "image"
)

type Model struct {
	Type    ModelType
	Name    string
	BaseUrl string
	ApiKey  string
}

// Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// for example, OpenAI: gpt-4o-mini, gpt-4o, gpt-o1
type Level int

const (
	L0 Level = iota
	L1
	L2
	L3
)

type LLMConfig struct {
	Model   string
	BaseUrl string
	ApiKey  string

	L1Model   string
	L1BaseUrl string
	L1ApiKey  string

	L2Model   string
	L2BaseUrl string
	L2ApiKey  string

	L3Model   string
	L3BaseUrl string
	L3ApiKey  string

	ImageModel   string
	ImageBaseUrl string
	ImageApiKey  string
}

func (cfg *LLMConfig) Clone() *LLMConfig {
	n := &LLMConfig{
		ApiKey:  cfg.ApiKey,
		BaseUrl: cfg.BaseUrl,
		Model:   cfg.Model,
	}
	return n
}

func Level1(cfg *LLMConfig) *Model {
	return CreateModel(cfg, L1)
}

func Level2(cfg *LLMConfig) *Model {
	return CreateModel(cfg, L2)
}

func Level3(cfg *LLMConfig) *Model {
	return CreateModel(cfg, L3)
}

func ImageModel(cfg *LLMConfig) *Model {
	if cfg == nil {
		return &Model{}
	}
	model := &Model{
		Type:    ModelTypeImage,
		Name:    cfg.ImageModel,
		BaseUrl: cfg.BaseUrl,
		ApiKey:  cfg.ApiKey,
	}
	if cfg.ImageApiKey != "" {
		model.ApiKey = cfg.ImageApiKey
	}
	if cfg.ImageBaseUrl != "" {
		model.BaseUrl = cfg.ImageBaseUrl
	}

	return model
}

// CreateModel creates a model with the given configuration and optional level
func CreateModel(cfg *LLMConfig, opt ...Level) *Model {
	if cfg == nil {
		return &Model{}
	}
	model := &Model{
		Type:    ModelTypeText,
		Name:    cfg.Model,
		BaseUrl: cfg.BaseUrl,
		ApiKey:  cfg.ApiKey,
	}

	// default level
	level := L0
	if len(opt) > 0 {
		level = opt[0]
	}

	switch level {
	case L0:
		return model
	case L1:
		if cfg.L1ApiKey != "" {
			model.ApiKey = cfg.L1ApiKey
		}
		if cfg.L1Model != "" {
			model.Name = cfg.L1Model
		}
		if cfg.L1BaseUrl != "" {
			model.BaseUrl = cfg.L1BaseUrl
		}
	case L2:
		if cfg.L2ApiKey != "" {
			model.ApiKey = cfg.L2ApiKey
		}
		if cfg.L2Model != "" {
			model.Name = cfg.L2Model
		}
		if cfg.L2BaseUrl != "" {
			model.BaseUrl = cfg.L2BaseUrl
		}
	case L3:
		if cfg.L3ApiKey != "" {
			model.ApiKey = cfg.L3ApiKey
		}
		if cfg.L3Model != "" {
			model.Name = cfg.L3Model
		}
		if cfg.L3BaseUrl != "" {
			model.BaseUrl = cfg.L3BaseUrl
		}
	}

	return model
}
