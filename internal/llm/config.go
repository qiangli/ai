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

	DBConfig *db.DBConfig
}
