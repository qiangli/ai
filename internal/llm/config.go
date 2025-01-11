package llm

import (
	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal/db"
)

type Config struct {
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

	Editor string

	// Current working directory where AI script is executed
	WorkDir string

	Interactive bool

	Clipin  bool
	Clipout bool
	Stdin   bool

	Me string

	MetaPrompt bool

	Command string
	Args    []string

	Tools []openai.ChatCompletionToolParam

	DBConfig *db.DBConfig `mapstructure:"db"`

	ConfigFile string

	// ai binary path
	CommandPath string

	//
	Git *GitConfig
}

func (cfg *Config) Clone() *Config {
	n := &Config{
		ApiKey:        cfg.ApiKey,
		BaseUrl:       cfg.BaseUrl,
		Model:         cfg.Model,
		Debug:         cfg.Debug,
		DryRun:        cfg.DryRun,
		DryRunContent: cfg.DryRunContent,
		Editor:        cfg.Editor,
		WorkDir:       cfg.WorkDir,
		Interactive:   cfg.Interactive,
		Clipin:        cfg.Clipin,
		Clipout:       cfg.Clipout,
		Stdin:         cfg.Stdin,
		Me:            cfg.Me,
		MetaPrompt:    cfg.MetaPrompt,
		Command:       cfg.Command,
		ConfigFile:    cfg.ConfigFile,
		Args:          nil,
		Tools:         nil,
		DBConfig:      nil,
	}

	// shallow copy
	n.Args = cfg.Args
	n.Tools = cfg.Tools
	n.DBConfig = cfg.DBConfig

	return n
}

type GitConfig struct {
	Short bool
}
