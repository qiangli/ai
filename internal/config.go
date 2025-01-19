package internal

import (
	_ "embed"
	"fmt"

	"github.com/openai/openai-go"
)

//go:embed ai.yaml
var configFileContent string

func GetDefaultConfig() string {
	return configFileContent
}

// global flags
var Debug bool // verbose output

var DryRun bool
var DryRunContent string

var WorkDir string

type AppConfig struct {
	ConfigFile string

	// ai binary path
	CommandPath string

	LLM *LLMConfig

	Role   string
	Prompt string

	Command string
	Args    []string

	Editor string

	Clipin  bool
	Clipout bool
	Stdin   bool

	Me string
}

type Tool openai.ChatCompletionToolParam

type Tools []openai.ChatCompletionToolParam

type LLMConfig struct {
	Workspace string

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

	Debug bool

	DryRun        bool
	DryRunContent string

	// Current working directory where AI script is executed
	WorkDir string

	Interactive bool

	MetaPrompt bool

	Tools Tools

	Output string

	//
	Git *GitConfig

	Sql *SQLConfig
}

func (cfg *LLMConfig) Clone() *LLMConfig {
	n := &LLMConfig{
		ApiKey:        cfg.ApiKey,
		BaseUrl:       cfg.BaseUrl,
		Model:         cfg.Model,
		Debug:         cfg.Debug,
		DryRun:        cfg.DryRun,
		DryRunContent: cfg.DryRunContent,
		WorkDir:       cfg.WorkDir,
		Interactive:   cfg.Interactive,
		MetaPrompt:    cfg.MetaPrompt,
		Tools:         nil,
		Sql:           nil,
		Git:           nil,
	}

	// shallow copy
	n.Tools = cfg.Tools
	n.Sql = cfg.Sql
	n.Git = cfg.Git

	return n
}

type GitConfig struct {
}

type SQLConfig struct {
	DBConfig *DBConfig `mapstructure:"db"`
}

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

	Content string
}

type Model struct {
	// Provider string

	Name    string
	BaseUrl string
	ApiKey  string

	Tools Tools

	DryRun        bool
	DryRunContent string
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

func Level1(cfg *LLMConfig) *Model {
	return CreateModel(cfg, L1)
}

func Level2(cfg *LLMConfig) *Model {
	return CreateModel(cfg, L2)
}

func Level3(cfg *LLMConfig) *Model {
	return CreateModel(cfg, L3)
}

// CreateModel creates a model with the given configuration and optional level
func CreateModel(cfg *LLMConfig, opt ...Level) *Model {
	model := &Model{
		Name:          cfg.Model,
		BaseUrl:       cfg.BaseUrl,
		ApiKey:        cfg.ApiKey,
		Tools:         cfg.Tools,
		DryRun:        cfg.DryRun,
		DryRunContent: cfg.DryRunContent,
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

type ToolConfig struct {
	Model    *Model
	DBConfig *DBConfig
}
