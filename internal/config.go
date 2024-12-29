package internal

type Config struct {
	ApiKey  string
	BaseUrl string
	Model   string

	Debug bool

	DryRun     bool
	DryRunFile string

	Editor string

	// Current working directory where AI script is executed
	WorkDir string
}
