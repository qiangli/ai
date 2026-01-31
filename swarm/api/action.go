package api

import (
	"context"
	"html/template"
	"maps"
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

// TODO split name into kit, name|pack/sub|command
// command line: [ACTION] [OPTIONS] MESSAGE...
type Action struct {
	// unique tool call identifier
	CallID string `json:"call_id"`

	// ID string `json:"id"`

	// agent/tool/command
	// agent:pack/sub
	// kit:name
	// /bin/comand
	Command string `json:"command"`

	// arguments including name
	Arguments Arguments `json:"arguments"`
}

func (r *Action) Kit() (string, string) {
	kit, name := Kitname(r.Command).Decode()
	return kit, name
}

type Arguments = ArgMap

func NewArguments() Arguments {
	return make(map[string]any)
}

// TODO get rid of this
func (r Arguments) Get2(key string) (any, bool) {
	v, ok := r[key]
	return v, ok
}

func (r Arguments) AddArgs(args map[string]any) Arguments {
	maps.Copy(r, args)
	return r
}

// clear all entries and copy args
// while maintaining the same old reference
func (r Arguments) SetArgs(args map[string]any) Arguments {
	for k := range r {
		delete(r, k)
	}
	maps.Copy(r, args)
	return r
}

// Return args specified by keys
func (r Arguments) GetArgs(keys []string) map[string]any {
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

// Copye all key/value pairs to dst
func (r Arguments) Copy(dst map[string]any) Arguments {
	maps.Copy(dst, r)
	return r
}

// openai: ChatCompletionMessageToolCallUnion
// genai: FunctionCall
// anthropic: ToolUseBlock
type ToolCall Action

func NewToolCall(cid string, cmd string, args map[string]any) *ToolCall {
	tc := &ToolCall{
		CallID:    cid,
		Command:   cmd,
		Arguments: args,
	}
	return tc
}

type ActionRunner interface {
	Run(context.Context, string, map[string]any) (any, error)
}

type App struct {
	// user id: auth email
	UserID string

	// workspace root. default: $HOME/.ai/
	Base string
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

// App config declares the default values for agent/tool/model
type AppConfig struct {
	// Top level field config not supported. these should be agent specific
	// Name        string
	// Display     string
	// Description string
	// Instruction string
	// Context     string
	// Message     string
	// Parameters  Parameters

	//
	Pack string `yaml:"pack" json:"pack"`
	// list of agents
	Agents []*AgentConfig `yaml:"agents" json:"agents"`

	// app level global vars
	Environment map[string]any `yaml:"environment" json:"environment"`

	// app level agent vars
	Arguments map[string]any `yaml:"arguments" json:"arguments"`

	// Kit specifies a namespace for the action
	// Action: agent, tool, and command
	//
	// Examples:
	// Local machine
	// File system
	// Container name
	// MCP server name
	// Internet
	Kit string `yaml:"kit" json:"kit"`

	// action type:
	// func, system, agent...
	Type string `yaml:"type" json:"type"`

	// list of tools
	Tools []*ToolConfig `yaml:"tools" json:"tools"`

	// modelset name
	Set string `yaml:"set" json:"set"`

	// set/level alias - not the LLM model
	Model string `yaml:"model" json:"model"`

	//
	MaxTurns int `yaml:"max_turns" json:"max_turns"`
	MaxTime  int `yaml:"max_time" json:"max_time"`

	// memory context
	MaxHistory int `yaml:"max_history" json:"max_history"`
	MaxSpan    int `yaml:"max_span" json:"max_span"`

	// logging: quiet | informative | verbose
	LogLevel string `yaml:"log_level" json:"log_level"`

	// model provider
	Provider string `yaml:"provider" json:"provider"`
	BaseUrl  string `yaml:"base_url" json:"base_url"`

	// api token lookup key - not the LLM api token
	ApiKey string `yaml:"api_key" json:"api_key"`

	Models map[string]*ModelConfig `yaml:"models" json:"models"`

	// The raw data for this config
	RawContent []byte `yaml:"-" json:"-"`

	// TODO for debugging
	Source string `yaml:"-" json:"-"`

	Store AssetStore `yaml:"-" json:"-"`
	// relative to store root
	BaseDir string `yaml:"-" json:"-"`
}

// ToMap converts AppConfig to a map[string]any
// map only the fileds common in action config
// and only for non zero values.
func (ac *AppConfig) ToMap() map[string]any {
	result := make(map[string]any)
	// agent/tool/model
	if ac.Pack != "" {
		result["pack"] = ac.Pack
	}
	if ac.Kit != "" {
		result["kit"] = ac.Kit
	}
	if ac.Type != "" {
		result["type"] = ac.Type
	}
	if ac.Set != "" {
		result["set"] = ac.Set
	}

	// common fields
	if ac.Model != "" {
		result["model"] = ac.Model
	}
	if ac.MaxTurns > 0 {
		result["max_turns"] = ac.MaxTurns
	}
	if ac.MaxTime > 0 {
		result["max_time"] = ac.MaxTime
	}
	if ac.MaxHistory > 0 {
		result["max_history"] = ac.MaxHistory
	}
	if ac.MaxSpan > 0 {
		result["max_span"] = ac.MaxSpan
	}
	//
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

//

// global/agent scope vars
type Environment struct {
	Env map[string]any `json:"env"`
	mu  sync.RWMutex   `json:"-"`
}

func NewEnvironment() *Environment {
	return &Environment{
		Env: make(map[string]any),
	}
}

// Return value for key
func (g *Environment) Get(key string) (any, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if v, ok := g.Env[key]; ok {
		return v, ok
	}
	return nil, false
}

// Return string value for key, empty if key it not set or value can not be converted to string
func (g *Environment) GetString(key string) string {
	if v, ok := g.Get(key); ok {
		return ToString(v)
	}
	return ""
}

// Return int value for key, 0 if key is not set or value can not be converted to int
func (g *Environment) GetInt(key string) int {
	if v, ok := g.Get(key); ok {
		return ToInt(v)
	}
	return 0
}

// Return all envs
func (g *Environment) GetAllEnvs() map[string]any {
	return g.GetEnvs(nil)
}

// Return envs specified by keys
func (g *Environment) GetEnvs(keys []string) map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	envs := make(map[string]any)
	if len(keys) == 0 {
		maps.Copy(envs, g.Env)
		return envs
	}
	for _, k := range keys {
		envs[k] = g.Env[k]
	}
	return envs
}

func (g *Environment) Set(key string, val any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Env[key] = val
}

// Clear all entries from the map and copy the new values
// while maintaining the same old reference.
func (g *Environment) SetEnvs(envs map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k := range g.Env {
		delete(g.Env, k)
	}
	maps.Copy(g.Env, envs)
}

func (g *Environment) Unset(key string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.Env, key)
}

func (g *Environment) UnsetEnvs(keys []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, k := range keys {
		delete(g.Env, k)
	}
}

// copy all src values to the environment env
func (g *Environment) AddEnvs(src map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	maps.Copy(g.Env, src)
}

// thread safe access to the env
func (g *Environment) Apply(fn func(map[string]any) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fn(g.Env)
}

// copy all environment env to dst
func (g *Environment) Copy(dst map[string]any) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	maps.Copy(dst, g.Env)
}

// agent/tool parameters
type Parameters map[string]any

func (r Parameters) Defaults() map[string]any {
	if len(r) == 0 {
		return nil
	}
	obj := r["properties"]
	props, _ := ToMap(obj)
	var data = make(map[string]any)
	for key, prop := range props {
		if p, ok := prop.(map[string]any); ok {
			if def, ok := p["default"]; ok {
				data[key] = def
			}
		}
	}
	return data
}
