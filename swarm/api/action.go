package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"maps"
	"strconv"
	"strings"
	"sync"
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
	ID string

	// agent/tool name
	Name string

	// arguments including name
	Arguments *Arguments
}

func NewAction(id string, name string, args map[string]any) *Action {
	return &Action{
		ID:   id,
		Name: name,
		Arguments: &Arguments{
			args: args,
		},
	}
}

type Arguments struct {
	args map[string]any
	mu   sync.RWMutex
}

func NewArguments() *Arguments {
	return &Arguments{
		args: make(map[string]any),
	}
}

func (r *Arguments) Query() string {
	return r.GetString("query")
}

func (r *Arguments) SetQuery(s string) {
	r.Set("query", s)
}

func (r *Arguments) Instruction() string {
	return r.GetString("instruction")
}

func (r *Arguments) SetInstruction(s string) {
	r.Set("instruction", s)
}

func (r *Arguments) Get(key string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.args[key]
	return v, ok
}

func (r *Arguments) GetString(key string) string {
	if v, ok := r.Get(key); ok {
		return ToString(v)
	}
	return ""
}

func (r *Arguments) GetInt(key string) int {
	if v, ok := r.Get(key); ok {
		return ToInt(v)
	}
	return 0
}

func (r *Arguments) Set(key string, data any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.args[key] = data
}

func (r *Arguments) SetArgs(args map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for k, v := range args {
		r.args[k] = v
	}
}

func (r *Arguments) GetAllArgs() map[string]any {
	return r.GetArgs(nil)
}

// Return args specified by keys
func (r *Arguments) GetArgs(keys []string) map[string]any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	args := make(map[string]any)
	if len(keys) == 0 {
		maps.Copy(args, r.args)
		return args
	}
	for _, k := range keys {
		args[k] = r.args[k]
	}
	return args
}

func (r *Arguments) Add(src map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	maps.Copy(r.args, src)
}

func (r *Arguments) Copy(dst map[string]any) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	maps.Copy(dst, r.args)
}

func (r *Arguments) Clone() *Arguments {
	r.mu.Lock()
	defer r.mu.Unlock()

	args := make(map[string]any)
	maps.Copy(args, r.args)
	return &Arguments{
		args: args,
	}
}

// openai: ChatCompletionMessageToolCallUnion
// genai: FunctionCall
// anthropic: ToolUseBlock
type ToolCall Action

func NewToolCall(id string, name string, args map[string]any) *ToolCall {
	tc := &ToolCall{
		ID:   id,
		Name: name,
		Arguments: &Arguments{
			args: args,
		},
	}
	return tc
}

type ActionRunner interface {
	Run(context.Context, string, map[string]any) (any, error)
}

type ActionConfig struct {
	// kit specifies a namespace for the action
	// examples:
	// class name
	// MCP server name
	// file system
	// container name
	// virtual machine name
	// tool/function (Gemini)
	Kit string `yaml:"kit"`

	// action type:
	// func, system, agent...
	Type string `yaml:"type"`

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

	// app level global vars
	Environment map[string]any `yaml:"environment"`
}

func (ac *ActionConfig) ToMap() map[string]any {
	result := map[string]any{
		"kit":  ac.Kit,
		"type": ac.Type,
		"name": ac.Name,
		// "arguments":    ac.Arguments,
		"message":     ac.Message,
		"instruction": ac.Instruction,
		"model":       ac.Model,
		"max_turns":   ac.MaxTurns,
		"max_time":    ac.MaxTime,
		"format":      ac.Format,
		"max_history": ac.MaxHistory,
		"max_span":    ac.MaxSpan,
		"context":     ac.Context,
		"log_level":   ac.LogLevel,
		// "environment":  ac.Environment,
	}
	return result
}

type AppConfig struct {
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
	Kit string `yaml:"kit"`

	// action type:
	// func, system, agent...
	Type string `yaml:"type"`

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

	// app level global vars
	Environment map[string]any `yaml:"environment"`

	//
	// unique identifier
	ID string `yaml:"-"`

	// app root. default: $HOME/.ai/
	Base string `yaml:"-"`

	// auth email
	User string `yaml:"-"`

	// workspace root. default: <base>/workspace
	Workspace string `yaml:"-"`

	Session string `yaml:"-"`

	// name of custom creator agent for this agent configuration
	Creator string `yaml:"creator"`

	// middleware chain
	Chain *ChainConfig `yaml:"chain"`

	// // action name and arguments
	// Name      string         `yaml:"name"`
	// Arguments map[string]any `yaml:"arguments"`

	// user message
	// Message string `yaml:"message"`

	// system prompt
	// Instruction string `yaml:"instruction"`

	// set/level key - not the LLM model
	// Model string `yaml:"model"`

	//
	Pack string `yaml:"pack"`

	Agents []*AgentConfig `yaml:"agents"`

	// //
	// MaxTurns int `yaml:"max_turns"`
	// MaxTime  int `yaml:"max_time"`

	// // output format: json | text
	// Format string `yaml:"format"`

	// // memory context
	// MaxHistory int    `yaml:"max_history"`
	// MaxSpan    int    `yaml:"max_span"`
	// Context    string `yaml:"context"`

	// // logging: quiet | informative | verbose
	// LogLevel string `yaml:"log_level"`

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
	// Kit string `yaml:"kit"`

	// action type:
	// func, system, agent...
	// Type  string        `yaml:"type"`
	Tools []*ToolConfig `yaml:"tools"`

	// modelset name
	Set    string                  `yaml:"set"`
	Models map[string]*ModelConfig `yaml:"models"`

	// // app level global vars
	// Environment map[string]any `yaml:"environment"`

	// TODO use arguments
	Clipin     bool
	ClipWait   bool
	Clipout    bool
	ClipAppend bool
	Stdin      bool
}

// ToMap converts AppConfig to a map[string]any
// map only the fileds common in action config.
func (ac *AppConfig) ToMap() map[string]any {
	return map[string]any{
		"kit":  ac.Kit,
		"type": ac.Type,
		"name": ac.Name,
		// "arguments":   ac.Arguments,
		"message":     ac.Message,
		"instruction": ac.Instruction,
		"model":       ac.Model,
		"max_turns":   ac.MaxTurns,
		"max_time":    ac.MaxTime,
		"format":      ac.Format,
		"max_history": ac.MaxHistory,
		"max_span":    ac.MaxSpan,
		"context":     ac.Context,
		"log_level":   ac.LogLevel,
		// // "environment": ac.Environment,
		// "id":          ac.ID,
		// "base":        ac.Base,
		// "user":        ac.User,
		// "workspace":   ac.Workspace,
		// "chat_id":     ac.ChatID,
		// "creator":     ac.Creator,
		// // "chain":       ac.Chain,
		// "pack":        ac.Pack,
		// // "agents":      ac.Agents,
		// "provider":    ac.Provider,
		// "base_url":    ac.BaseUrl,
		// "api_key":     ac.ApiKey,
		// // "tools":       ac.Tools,
		// "set":         ac.Set,
		// // "models":      ac.Models,
		// // "clipin":      ac.Clipin,
		// // "clipwait":    ac.ClipWait,
		// // "clipout":     ac.Clipout,
		// // "clipappend":  ac.ClipAppend,
		// // "stdin":       ac.Stdin,
	}
}

// // Clone is a shallow copy of member fields of the configration
// func (cfg *ActionConfig) Clone() *ActionConfig {
// 	return &ActionConfig{
// 		Kit:  cfg.Kit,
// 		Type: cfg.Type,
// 		//
// 		Name:      cfg.Name,
// 		Arguments: cfg.Arguments,
// 		Model:     cfg.Model,
// 		//
// 		Message:     cfg.Message,
// 		Instruction: cfg.Instruction,
// 		// Editor:     cfg.Editor,
// 		// Clipin:     cfg.Clipin,
// 		// ClipWait:   cfg.ClipWait,
// 		// Clipout:    cfg.Clipout,
// 		// ClipAppend: cfg.ClipAppend,
// 		// IsPiped:    cfg.IsPiped,
// 		// Stdin: cfg.Stdin,
// 		//
// 		Format: cfg.Format,

// 		MaxHistory: cfg.MaxHistory,
// 		MaxSpan:    cfg.MaxSpan,
// 		Context:    cfg.Context,
// 		//
// 		LogLevel: cfg.LogLevel,

// 		// Base:      cfg.Base,
// 		// Workspace: cfg.Workspace,

// 		MaxTime:  cfg.MaxTime,
// 		MaxTurns: cfg.MaxTurns,
// 		//
// 		Environment: cfg.Environment,
// 	}
// }

// type AppConfig ActionConfig

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

func (cfg *AppConfig) IsStdin() bool {
	return cfg.Stdin
}

func (cfg *AppConfig) IsClipin() bool {
	return cfg.Clipin
}

func (cfg *AppConfig) IsSpecial() bool {
	return cfg.IsStdin() || cfg.IsClipin()
}

func (cfg *AppConfig) HasInput() bool {
	return cfg.Message != "" || cfg.Name != ""
}

func (cfg *AppConfig) Interactive() bool {
	_, ok := cfg.Arguments["interactive"]
	return ok
}

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

func ToString(data any) string {
	if data == nil {
		return ""
	}
	if s, ok := data.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", data)
}

func ToInt(data any) int {
	if data == nil {
		return 0
	}
	if i, ok := data.(int); ok {
		return i
	}
	if i, ok := data.(int8); ok {
		return int(i)
	}
	if i, ok := data.(int16); ok {
		return int(i)
	}
	if i, ok := data.(int32); ok {
		return int(i)
	}
	if i, ok := data.(int64); ok {
		return int(i)
	}
	if s, ok := data.(string); ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			return i
		}
	}
	return 0
}
