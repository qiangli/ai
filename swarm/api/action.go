package api

import (
	"context"
	"encoding/base64"
	"fmt"
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

type Action struct {
	// unique identifier
	ID string `json:"-"`

	// agent/tool name
	Name string `json:"name"`

	// arguments including name
	Arguments map[string]any `json:"arguments"`
}

// openai: ChatCompletionMessageToolCallUnion
// genai: FunctionCall
// anthropic: ToolUseBlock
type ToolCall Action

type ActionRunner interface {
	Run(context.Context, string, map[string]any) (any, error)
}

type AppConfig struct {
	// unique identifier
	ID string `yaml:"-"`

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

	//
	Pack string `yaml:"pack"`

	Agents []*AgentConfig `yaml:"agents"`

	//
	MaxTurns int `yaml:"max_turns"`
	MaxTime  int `yaml:"max_time"`

	// output format: json | text
	Format string `yaml:"format"`

	// memory context
	MaxHistory int    `yaml:"max_history"`
	MaxSpan    int    `yaml:"max_span"`
	Context    string `yaml:"context"`

	// logging: quiet | informative | verbose
	LogLevel string `yaml:"log_level"`

	// tool / model provider
	Provider string `yaml:"provider"`
	BaseUrl  string `yaml:"base_url"`

	// api token lookup key
	ApiKey string `yaml:"api_key"`

	// tool kit

	// kit specifies a namespace
	//
	// class name
	// MCP server name
	// file system
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

func ToResult(data any) *Result {
	if v, ok := data.(*Result); ok {
		if len(v.Content) == 0 {
			return v
		}
		if v.MimeType == ContentTypeImageB64 {
			return v
		}
		if strings.HasPrefix(v.MimeType, "text/") {
			return &Result{
				MimeType: v.MimeType,
				Value:    string(v.Content),
			}
		}
		return &Result{
			MimeType: v.MimeType,
			Value:    dataURL(v.MimeType, v.Content),
		}
		// // image
		// // transform media response into data url
		// presigned, err := sw.save(sw)
		// if err != nil {
		// 	return &api.Result{
		// 		Value: err.Error(),
		// 	}
		// }

		// return &api.Result{
		// 	MimeType: v.MimeType,
		// 	Value:    presigned,
		// }
	}
	if s, ok := data.(string); ok {
		return &Result{
			Value: s,
		}
	}
	return &Result{
		Value: fmt.Sprintf("%v", data),
	}
}

// https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data
// data:[<media-type>][;base64],<data>
func dataURL(mime string, raw []byte) string {
	encoded := base64.StdEncoding.EncodeToString(raw)
	d := fmt.Sprintf("data:%s;base64,%s", mime, encoded)
	return d
}
