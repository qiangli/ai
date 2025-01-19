package llm

// import (
// 	"github.com/openai/openai-go"
// 	"github.com/qiangli/ai/internal/db"
// )

// type Tool openai.ChatCompletionToolParam

// type Tools []openai.ChatCompletionToolParam

// type Config struct {
// 	Workspace string

// 	Model   string
// 	BaseUrl string
// 	ApiKey  string

// 	L1Model   string
// 	L1BaseUrl string
// 	L1ApiKey  string

// 	L2Model   string
// 	L2BaseUrl string
// 	L2ApiKey  string

// 	L3Model   string
// 	L3BaseUrl string
// 	L3ApiKey  string

// 	Debug bool

// 	DryRun        bool
// 	DryRunContent string

// 	Editor string

// 	// Current working directory where AI script is executed
// 	WorkDir string

// 	Interactive bool

// 	Clipin  bool
// 	Clipout bool
// 	Stdin   bool

// 	Me string

// 	MetaPrompt bool

// 	AgentCommand string
// 	// Args    []string

// 	Tools Tools

// 	ConfigFile string

// 	// ai binary path
// 	CommandPath string

// 	Output string

// 	//
// 	Git *GitConfig

// 	Sql *SQLConfig
// }

// func (cfg *Config) Clone() *Config {
// 	n := &Config{
// 		ApiKey:        cfg.ApiKey,
// 		BaseUrl:       cfg.BaseUrl,
// 		Model:         cfg.Model,
// 		Debug:         cfg.Debug,
// 		DryRun:        cfg.DryRun,
// 		DryRunContent: cfg.DryRunContent,
// 		Editor:        cfg.Editor,
// 		WorkDir:       cfg.WorkDir,
// 		Interactive:   cfg.Interactive,
// 		Clipin:        cfg.Clipin,
// 		Clipout:       cfg.Clipout,
// 		Stdin:         cfg.Stdin,
// 		Me:            cfg.Me,
// 		MetaPrompt:    cfg.MetaPrompt,
// 		Command:       cfg.Command,
// 		ConfigFile:    cfg.ConfigFile,
// 		Args:          nil,
// 		Tools:         nil,
// 		Sql:           nil,
// 		Git:           nil,
// 	}

// 	// shallow copy
// 	n.Args = cfg.Args
// 	n.Tools = cfg.Tools
// 	n.Sql = cfg.Sql
// 	n.Git = cfg.Git

// 	return n
// }

// type GitConfig struct {
// 	Short bool
// }

// type SQLConfig struct {
// 	DBConfig *db.DBConfig `mapstructure:"db"`
// }

// type Message struct {
// 	Role   string
// 	Prompt string
// 	Model  *Model

// 	Input   string
// 	DBCreds *db.DBConfig

// 	Content string
// }

// type Model struct {
// 	// Provider string

// 	Name    string
// 	BaseUrl string
// 	ApiKey  string

// 	Tools Tools

// 	DryRun        bool
// 	DryRunContent string
// }

// // Level represents the "intelligence" level of the model. i.e. basic, regular, advanced
// // for example, OpenAI: gpt-4o-mini, gpt-4o, gpt-o1
// type Level int

// const (
// 	L0 Level = iota
// 	L1
// 	L2
// 	L3
// )

// func Level1(cfg *Config) *Model {
// 	return CreateModel(cfg, L1)
// }

// func Level2(cfg *Config) *Model {
// 	return CreateModel(cfg, L2)
// }

// func Level3(cfg *Config) *Model {
// 	return CreateModel(cfg, L3)
// }

// // CreateModel creates a model with the given configuration and optional level
// func CreateModel(cfg *Config, opt ...Level) *Model {
// 	model := &Model{
// 		Name:          cfg.Model,
// 		BaseUrl:       cfg.BaseUrl,
// 		ApiKey:        cfg.ApiKey,
// 		Tools:         cfg.Tools,
// 		DryRun:        cfg.DryRun,
// 		DryRunContent: cfg.DryRunContent,
// 	}

// 	// default level
// 	level := L0
// 	if len(opt) > 0 {
// 		level = opt[0]
// 	}

// 	switch level {
// 	case L0:
// 		return model
// 	case L1:
// 		if cfg.L1ApiKey != "" {
// 			model.ApiKey = cfg.L1ApiKey
// 		}
// 		if cfg.L1Model != "" {
// 			model.Name = cfg.L1Model
// 		}
// 		if cfg.L1BaseUrl != "" {
// 			model.BaseUrl = cfg.L1BaseUrl
// 		}
// 	case L2:
// 		if cfg.L2ApiKey != "" {
// 			model.ApiKey = cfg.L2ApiKey
// 		}
// 		if cfg.L2Model != "" {
// 			model.Name = cfg.L2Model
// 		}
// 		if cfg.L2BaseUrl != "" {
// 			model.BaseUrl = cfg.L2BaseUrl
// 		}
// 	case L3:
// 		if cfg.L3ApiKey != "" {
// 			model.ApiKey = cfg.L3ApiKey
// 		}
// 		if cfg.L3Model != "" {
// 			model.Name = cfg.L3Model
// 		}
// 		if cfg.L3BaseUrl != "" {
// 			model.BaseUrl = cfg.L3BaseUrl
// 		}
// 	}

// 	return model
// }

// type ToolConfig struct {
// 	Model    *Model
// 	DBConfig *db.DBConfig
// }
