package api

import (
	"context"
	"html/template"
	"strings"
)

type State int

const (
	StateExit State = iota

	StateTransfer
	StateInputWait
	StateToolCall
)

func (r State) String() string {
	switch r {
	case StateExit:
		return "EXIT"
	case StateTransfer:
		return "TRANSFER"
	case StateInputWait:
		return "INPUT_WAIT"
	case StateToolCall:
		return "TOOL_CALL"
	}
	return "DEFAULT"
}

func (r State) Equal(s string) bool {
	return strings.ToUpper(s) == r.String()
}

func ParseState(s string) State {
	switch strings.ToUpper(s) {
	case "EXIT":
		return StateExit
	case "TRANSFER":
		return StateTransfer
	case "INPUT_WAIT":
		return StateInputWait
	case "TOOL_CALL":
		return StateToolCall
	}
	return StateExit
}

type TemplateFuncMap = template.FuncMap

type ToolRunner func(context.Context, string, map[string]any) (*Result, error)

type Action struct {
	ID string `json:"id"`

	// agent/tool name
	Name string `json:"name"`

	// arguments including name
	Arguments map[string]any `json:"arguments"`

	// //
	// Tool  *ToolFunc `json:"-"`
	// Agent *Agent    `json:"-"`
}

// openai: ChatCompletionMessageToolCallUnion
// genai: FunctionCall
// anthropic: ToolUseBlock
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type AppConfig struct {
	// app root. default: $HOME/.ai/
	Base string `yaml:"-"`

	// auth email
	User string `yaml:"-"`

	// workspace root. default: <base>/workspace
	Workspace string `yaml:"-"`

	ChatID string `yaml:"-"`

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

	// memory context
	// max history: 0 max span: 0
	// New        *bool  `yaml:"new,omitempty"`
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
	Type  string        `yaml:"type"`
	Tools []*ToolConfig `yaml:"tools"`

	// modelset name
	Set    string                  `yaml:"set"`
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
		Name:      cfg.Name,
		Arguments: cfg.Arguments,
		Model:     cfg.Model,
		//
		Message:     cfg.Message,
		Instruction: cfg.Instruction,
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
		Base:      cfg.Base,
		Workspace: cfg.Workspace,
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

type AgentTool AppConfig
