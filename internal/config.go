package internal

import (
	_ "embed"

	"github.com/qiangli/ai/internal/api"
)

type DBConfig = api.DBCred
type LLMConfig = api.LLMConfig

type GitConfig struct {
}

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

	Debug bool

	// Current working directory where AI script is executed
	WorkDir     string
	Interactive bool
	MetaPrompt  bool

	MaxTime  int
	MaxTurns int
}
