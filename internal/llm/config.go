package llm

import (
	"github.com/openai/openai-go"
	"github.com/qiangli/ai/internal/db"
)

type Config struct {
	ApiKey  string
	BaseUrl string
	Model   string

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
