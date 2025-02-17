package internal

import (
	_ "embed"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal/api"
)

//go:embed ai.yaml
var configFileContent string

func GetDefaultConfig() string {
	return configFileContent
}

// global flags

var DryRun bool
var DryRunContent string

type AppConfig struct {
	ConfigFile string

	// ai binary path
	CommandPath string

	LLM *LLMConfig

	Git *GitConfig
	Db  *DBConfig

	Role   string
	Prompt string

	Command string
	Args    []string

	// --message takes precedence over all other forms of input
	Message string

	Editor string

	Clipin  bool
	Clipout bool
	Stdin   bool

	Files []string

	// Output format: raw or markdown
	Format string

	// Save output to file
	Output string

	Me string

	//
	Template string

	Workspace string
}

type Tool openai.ChatCompletionToolParam

type Tools []openai.ChatCompletionToolParam

type LLMConfig struct {
	// Workspace string

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

	Debug bool

	// Current working directory where AI script is executed
	WorkDir string

	Interactive bool

	MetaPrompt bool
}

func (cfg *LLMConfig) Clone() *LLMConfig {
	n := &LLMConfig{
		ApiKey:      cfg.ApiKey,
		BaseUrl:     cfg.BaseUrl,
		Model:       cfg.Model,
		Debug:       cfg.Debug,
		WorkDir:     cfg.WorkDir,
		Interactive: cfg.Interactive,
		MetaPrompt:  cfg.MetaPrompt,
	}

	return n
}

type GitConfig struct {
}

// type SQLConfig struct {
// 	DBConfig *DBConfig `mapstructure:"db"`
// }

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"name"`
}

// DSN returns the data source name for connecting to the database.
func (d *DBConfig) DSN() string {
	host := d.Host
	if host == "" {
		host = "localhost"
	}
	port := d.Port
	if port == "" {
		port = "5432"
	}
	dbname := d.DBName
	if dbname == "" {
		dbname = "postgres"
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, d.Username, d.Password, dbname)
}

func (d *DBConfig) IsValid() bool {
	return d.Username != "" && d.Password != ""
}

type Message struct {
	Role   string
	Prompt string
	Model  *Model

	Input   string
	DBCreds *DBConfig

	// Response
	Content string

	Next api.Action
}

type Model = api.Model
type Level = api.Level

func Level1(cfg *LLMConfig) *Model {
	return CreateModel(cfg, api.L1)
}

func Level2(cfg *LLMConfig) *Model {
	return CreateModel(cfg, api.L2)
}

func Level3(cfg *LLMConfig) *Model {
	return CreateModel(cfg, api.L3)
}

func ImageModel(cfg *LLMConfig) *Model {
	model := &Model{
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
	model := &Model{
		Name:    cfg.Model,
		BaseUrl: cfg.BaseUrl,
		ApiKey:  cfg.ApiKey,
	}

	// default level
	level := api.L0
	if len(opt) > 0 {
		level = opt[0]
	}

	switch level {
	case api.L0:
		return model
	case api.L1:
		if cfg.L1ApiKey != "" {
			model.ApiKey = cfg.L1ApiKey
		}
		if cfg.L1Model != "" {
			model.Name = cfg.L1Model
		}
		if cfg.L1BaseUrl != "" {
			model.BaseUrl = cfg.L1BaseUrl
		}
	case api.L2:
		if cfg.L2ApiKey != "" {
			model.ApiKey = cfg.L2ApiKey
		}
		if cfg.L2Model != "" {
			model.Name = cfg.L2Model
		}
		if cfg.L2BaseUrl != "" {
			model.BaseUrl = cfg.L2BaseUrl
		}
	case api.L3:
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

// type ToolConfig struct {
// 	Model    *Model
// 	DBConfig *DBConfig

// 	Next api.Action
// }
