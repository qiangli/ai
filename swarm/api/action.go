package api

import (
	"context"
	"html/template"
	"maps"
	"strings"
	// "sync"
)

type State int

const (
	StateExit State = iota

	// TODO terminate state for early tool call termination/cancclation
	StateTransfer
	StateInputWait
	StateToolCall
)

func (s State) String() string {
	switch s {
	case StateExit:
		return "EXIT"
	case StateTransfer:
		return "TRANSFER"
	case StateInputWait:
		return "INPUT_WAIT"
	case StateToolCall:
		return "TOOL_CALL"
	}
	return "EXIT"
}

func (s State) Equal(state string) bool {
	return strings.ToUpper(state) == s.String()
}

func ParseState(state string) State {
	switch strings.ToUpper(state) {
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
	ID string `json:"id"`

	// agent/tool name
	Name string `json:"name"`

	// arguments including name
	Arguments Arguments `json:"arguments"`
}

func NewAction(id string, name string, args map[string]any) *Action {
	return &Action{
		ID:   id,
		Name: name,
		// Arguments: &Arguments{
		// 	Args: args,
		// },
		Arguments: args,
	}
}

// type Arguments struct {
// 	Args ArgMap `json:"args"`

// 	// TODO redesign - comment out for now
// 	// mu   sync.RWMutex   `json:"-"`
// }

type Arguments ArgMap

// func NewArguments() *Arguments {
// 	return &Arguments{
// 		Args:  make(map[string]any),
// 	}
// }

func NewArguments() Arguments {
	return make(map[string]any)
}

func (r Arguments) Message() string {
	return r.GetString("message")
}

func (r Arguments) SetMessage(s any) Arguments {
	r.Set("message", s)
	return r
}

func (r Arguments) Get(key string) (any, bool) {
	// r.mu.RLock()
	// defer r.mu.RUnlock()
	v, ok := r[key]
	return v, ok
}

func (r Arguments) GetString(key string) string {
	if v, ok := r.Get(key); ok {
		return ToString(v)
	}
	return ""
}

func (r Arguments) GetInt(key string) int {
	if v, ok := r.Get(key); ok {
		return ToInt(v)
	}
	return 0
}

func (r Arguments) GetStringSlice(key string) []string {
	return r.GetStringSlice(key)
}

func (r Arguments) Set(key string, val any) Arguments {
	// r.mu.Lock()
	// defer r.mu.Unlock()
	r[key] = val
	return r
}

func (r Arguments) AddArgs(args map[string]any) Arguments {
	// r.mu.Lock()
	// defer r.mu.Unlock()
	maps.Copy(r, args)
	return r
}

// clear all entries and copy args
// while maintaining the same old reference
func (r Arguments) SetArgs(args map[string]any) Arguments {
	// r.mu.Lock()
	// defer r.mu.Unlock()
	for k := range r {
		delete(r, k)
	}
	maps.Copy(r, args)
	return r
}

func (r Arguments) GetAllArgs() map[string]any {
	return r.GetArgs(nil)
}

// Return args specified by keys
func (r Arguments) GetArgs(keys []string) map[string]any {
	// r.mu.RLock()
	// defer r.mu.RUnlock()
	args := make(map[string]any)
	if len(keys) == 0 {
		maps.Copy(args, r)
		return args
	}
	for _, k := range keys {
		args[k] = r[k]
	}
	return args
}

func (r Arguments) Copy(dst map[string]any) Arguments {
	// r.mu.RLock()
	// defer r.mu.RUnlock()
	maps.Copy(dst, r)
	return r
}

func (r Arguments) Clone() Arguments {
	// r.mu.Lock()
	// defer r.mu.Unlock()

	args := make(map[string]any)
	maps.Copy(args, r)
	// return &Arguments{
	// 	Args: args,
	// }
	return args
}

// openai: ChatCompletionMessageToolCallUnion
// genai: FunctionCall
// anthropic: ToolUseBlock
type ToolCall Action

func NewToolCall(id string, name string, args map[string]any) *ToolCall {
	tc := &ToolCall{
		ID:   id,
		Name: name,
		// Arguments: &Arguments{
		// 	Args: args,
		// },
		Arguments: args,
	}
	return tc
}

type ActionRunner interface {
	Run(context.Context, string, map[string]any) (any, error)
}

type App struct {
	// app root. default: $HOME/.ai/
	Base string

	// auth email
	User string

	Session string
}

type InputConfig struct {
	Message string
	Args    []string

	Clipin     bool
	ClipWait   bool
	Clipout    bool
	ClipAppend bool
	Stdin      bool
}

type AppConfig struct {
	// entry action
	// kit:name
	// pack/sub
	// default: pack[/pack]
	Action string `yaml:"action" json:"action"`

	// ActionConfig
	//
	// kit specifies a namespace for the action
	// examples:
	// class name
	// MCP server name
	// file system
	// container name
	// virtual machine name
	// tool/function (Gemini)
	Kit string `yaml:"kit" json:"kit"`

	// action type:
	// func, system, agent...
	Type string `yaml:"type" json:"type"`

	// action name and arguments
	Name      string         `yaml:"name" json:"name"`
	Arguments map[string]any `yaml:"arguments" json:"arguments"`

	// user message
	Message string `yaml:"message" json:"message"`

	// system prompt
	Instruction string `yaml:"instruction" json:"instruction"`

	// set/level key - not the LLM model
	Model string `yaml:"model" json:"model"`

	//
	MaxTurns int `yaml:"max_turns" json:"max_turns"`
	MaxTime  int `yaml:"max_time" json:"max_time"`

	// output format: json | text
	Format string `yaml:"format" json:"format"`

	// memory context
	MaxHistory int    `yaml:"max_history" json:"max_history"`
	MaxSpan    int    `yaml:"max_span" json:"max_span"`
	Context    string `yaml:"context" json:"context"`

	// logging: quiet | informative | verbose
	LogLevel string `yaml:"log_level" json:"log_level"`

	// app level global vars
	Environment map[string]any `yaml:"environment" json:"environment"`

	//
	Pack string `yaml:"pack" json:"pack"`

	Agents []*AgentConfig `yaml:"agents" json:"agents"`

	// tool / model provider
	Provider string `yaml:"provider" json:"provider"`
	BaseUrl  string `yaml:"base_url" json:"base_url"`

	// api token lookup key
	ApiKey string `yaml:"api_key" json:"api_key"`

	// action type:
	// func, system, agent...
	// Type  string        `yaml:"type"`
	Tools []*ToolConfig `yaml:"tools" json:"tools"`

	// modelset name
	Set    string                  `yaml:"set" json:"set"`
	Models map[string]*ModelConfig `yaml:"models" json:"models"`

	// The raw data for this config
	RawContent []byte `yaml:"-" json:"-"`

	// TODO for debugging
	// Source     string `yaml:"-" json:"-"`
}

// ToMap converts AppConfig to a map[string]any
// map only the fileds common in action config
// and only for non zero values.
func (ac *AppConfig) ToMap() map[string]any {
	result := make(map[string]any)
	if ac.Pack != "" {
		result["pack"] = ac.Pack
	}
	if ac.Action != "" {
		result["action"] = ac.Action
	}
	if ac.Kit != "" {
		result["kit"] = ac.Kit
	}
	if ac.Type != "" {
		result["type"] = ac.Type
	}
	if ac.Name != "" {
		result["name"] = ac.Name
	}
	// if ac.Message != "" {
	// 	result["message"] = ac.Message
	// }
	// if ac.Instruction != "" {
	// 	result["instruction"] = ac.Instruction
	// }
	// if ac.Context != "" {
	// 	result["context"] = ac.Context
	// }
	if ac.Model != "" {
		result["model"] = ac.Model
	}
	if ac.MaxTurns > 0 {
		result["max_turns"] = ac.MaxTurns
	}
	if ac.MaxTime > 0 {
		result["max_time"] = ac.MaxTime
	}
	if ac.Format != "" {
		result["format"] = ac.Format
	}
	if ac.MaxHistory > 0 {
		result["max_history"] = ac.MaxHistory
	}
	if ac.MaxSpan > 0 {
		result["max_span"] = ac.MaxSpan
	}
	if ac.LogLevel != "" {
		result["log_level"] = ac.LogLevel
	}

	return result
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

func (cfg *AppConfig) HasInput() bool {
	return cfg.Message != ""
}

func (cfg *AppConfig) Interactive() bool {
	_, ok := cfg.Arguments["interactive"]
	return ok
}
