package internal

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

	Command string
	Args    []string
}
