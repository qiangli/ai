package api

type Action struct {
	// agent/tool name
	Name string

	// arguments including name
	Arguments map[string]any

	//
	Tool  *ToolFunc
	Agent *Agent
}

type AppConfig struct {
	// app root. default: $HOME/.ai/
	Base string `yaml:"base"`

	// email
	User string `yaml:"user"`

	// workspace root. default: <base>/workspace
	Workspace string

	// action name and arguments
	Name      string         `yaml:"name"`
	Arguments map[string]any `yaml:"arguments"`

	// user message
	Message string `yaml:"message"`

	// system prompt
	Instruction string `yaml:"instruction"`

	// set/level key - not the LLM model
	Model string `yaml:"model"`

	Agents []*AgentConfig `yaml:"agents"`

	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory
	// max history: 0 max span: 0
	New        *bool  `yaml:"new,omitempty"`
	MaxHistory int    `yaml:"max_history"`
	MaxSpan    int    `yaml:"max_span"`
	Context    string `yaml:"context"`

	// logging: quiet | informative | verbose
	LogLevel string `yaml:"log_level"`

	// tool or model provider
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`

	// api token lookup key
	ApiKey string `yaml:"api_key"`

	// toolkit
	// kit:any
	// Kit   string        `yaml:"kit"`
	// Type  string        `yaml:"type"`
	// Tools []*ToolConfig `yaml:"tools"`

	// tools

	// kit name specifies a namespace.
	// e.g. but not limited to:
	// class name
	// MCP server name
	// virtual filesystem name
	// container name
	// virtual machine name
	// tool/function (Gemini)
	Kit string `yaml:"kit"`

	// action type:
	// func, system, agent...
	Type string `yaml:"type"`

	// // provider
	// Provider string `yaml:"provider"`
	// BaseUrl  string `yaml:"base_url"`
	// // name of api key
	// ApiKey string `yaml:"api_key"`

	// Connector *ConnectorConfig `yaml:"connector"`

	// system commands used by tools
	// Commands []string `yaml:"commands"`

	Tools []*ToolConfig `yaml:"tools"`

	// model set
	// set/any
	// Set    string                  `yaml:"set"`
	// Models map[string]*ModelConfig `yaml:"models"`

	// TODO separate?
	// model/tool shared default values
	// Provider string `yaml:"provider"`
	// BaseUrl  string `yaml:"base_url"`
	// api lookup key
	// ApiKey string `yaml:"api_key"`

	// model

	// model set name
	Set string `yaml:"set"`

	// // provider
	// Provider string `yaml:"provider"`
	// BaseUrl  string `yaml:"base_url"`
	// // name of api lookup key - never the actual api token
	// ApiKey string `yaml:"api_key"`

	Models map[string]*ModelConfig `yaml:"models"`

	// app level global vars
	Environment map[string]any `yaml:"environment"`

	// TODO use arguments
	Clipin     bool
	ClipWait   bool
	Clipout    bool
	ClipAppend bool
	Stdin      bool
}

// Clone is a shallow copy of member fields of the configration
func (cfg *AppConfig) Clone() *AppConfig {
	return &AppConfig{
		// Version:    cfg.Version,
		// ConfigFile: cfg.ConfigFile,
		//
		// AgentResource: cfg.AgentResource,
		//
		Name:  cfg.Name,
		Model: cfg.Model,
		//
		// Args:       cfg.Args,
		Message: cfg.Message,
		// Editor:     cfg.Editor,
		// Clipin:     cfg.Clipin,
		// ClipWait:   cfg.ClipWait,
		// Clipout:    cfg.Clipout,
		// ClipAppend: cfg.ClipAppend,
		// IsPiped:    cfg.IsPiped,
		// Stdin: cfg.Stdin,
		//
		Format: cfg.Format,
		// Output: cfg.Output,
		//
		// ChatID:     cfg.ChatID,
		// New:        cfg.New,
		MaxHistory: cfg.MaxHistory,
		MaxSpan:    cfg.MaxSpan,
		Context:    cfg.Context,
		//
		LogLevel: cfg.LogLevel,
		//
		// Unsafe:      cfg.Unsafe,
		// Base:        cfg.Base,
		// Workspace:   cfg.Workspace,
		// Interactive: cfg.Interactive,
		// Editing:     cfg.Editing,
		// Shell:       cfg.Shell,
		// Watch:       cfg.Watch,
		// ClipWatch:   cfg.ClipWatch,
		MaxTime:  cfg.MaxTime,
		MaxTurns: cfg.MaxTurns,
		// Stdout:      cfg.Stdout,
		// Stderr:      cfg.Stderr,
		// //
		// DryRun:        cfg.DryRun,
		// DryRunContent: cfg.DryRunContent,
	}
}

func (cfg *AppConfig) IsNew() bool {
	// return cfg.New != nil && *cfg.New
	return cfg.MaxHistory == 0
}

func (cfg *AppConfig) IsQuiet() bool {
	return ToLogLevel(cfg.LogLevel) == Quiet
}

func (cfg *AppConfig) IsInformative() bool {
	return ToLogLevel(cfg.LogLevel) == Informative
}

func (cfg *AppConfig) IsVerbose() bool {
	return ToLogLevel(cfg.LogLevel) == Verbose
}

func (cfg *AppConfig) IsTracing() bool {
	return ToLogLevel(cfg.LogLevel) == Tracing
}

func (r *AppConfig) IsStdin() bool {
	return r.Stdin
}

func (r *AppConfig) IsClipin() bool {
	return r.Clipin
}

func (r *AppConfig) IsSpecial() bool {
	return r.IsStdin() || r.IsClipin()
}

func (r *AppConfig) HasInput() bool {
	return r.Message != "" || r.Name != ""
}

func (cfg *AppConfig) Interactive() bool {
	_, ok := cfg.Arguments["interactive"]
	return ok
}

// agent or tool
// TODO redesign: agent <-> tool same?
type AgentTool struct {
	Agent *Agent
	Tool  *ToolFunc

	//
	Owner       string
	Instruction string
	Message     string
	Arguments   map[string]any
}
