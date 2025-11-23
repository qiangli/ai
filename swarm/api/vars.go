package api

import (
	"encoding/json"
	// "fmt"
	"maps"
	"sync"
)

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

type Environment struct {
	env map[string]any
	mu  sync.RWMutex
}

func NewEnvironment() *Environment {
	return &Environment{
		env: make(map[string]any),
	}
}

func (g *Environment) Get(key string) (any, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if v, ok := g.env[key]; ok {
		return v, ok
	}
	return nil, false
}

func (g *Environment) GetAllEnvs() map[string]any {
	return g.GetEnvs(nil)
}

// Return envs specified by keys
func (g *Environment) GetEnvs(keys []string) map[string]any {
	g.mu.RLock()
	defer g.mu.RUnlock()
	envs := make(map[string]any)
	if len(keys) == 0 {
		maps.Copy(envs, g.env)
		return envs
	}
	for _, k := range keys {
		envs[k] = g.env[k]
	}
	return envs
}

func (g *Environment) Set(key string, val any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.env[key] = val
}

// Clear all entries from the map and copy the new values
// while maintaining the same old reference.
func (g *Environment) SetEnvs(envs map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k := range g.env {
		delete(g.env, k)
	}
	maps.Copy(g.env, envs)
}

func (g *Environment) UnsetEnvs(keys []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, k := range keys {
		delete(g.env, k)
	}
}

// copy all src values to the environment env
func (g *Environment) AddEnvs(src map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	maps.Copy(g.env, src)
}

// thread safe access to the env
func (g *Environment) Apply(fn func(map[string]any) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fn(g.env)
}

// copy all environment env to dst
func (g *Environment) Copy(dst map[string]any) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	maps.Copy(dst, g.env)
}

func (g *Environment) Clone() *Environment {
	g.mu.Lock()
	defer g.mu.Unlock()

	env := make(map[string]any)
	maps.Copy(env, g.env)
	return &Environment{
		env: env,
	}
}

// global context
type Vars struct {
	// LogLevel LogLevel `json:"log_level"`

	// ID string `json:"chat_id"`
	// MaxTurns   int    `json:"max_turns"`
	// MaxTime    int    `json:"max_time"`
	// New        *bool  `json:"new"`
	// MaxHistory int    `json:"max_history"`
	// MaxSpan    int    `json:"max_span"`
	// Context    string `json:"context"`
	// Format     string `json:"format"`
	// Models     string `json:"models"`

	// Unsafe bool `json:"unsafe"`
	// Workspace string `json:"workspace"`

	// DryRun        bool   `json:"-"`
	// DryRunContent string `json:"-"`

	Global *Environment `json:"-"`

	// conversation history
	history []*Message `json:"-"`

	// initial size of hisotry
	// initLen int `json:"-"`

	toolcallHistory []*ToolCallEntry `json:"-"`

	mu sync.RWMutex
}

func (v *Vars) AddToolCall(item *ToolCallEntry) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.toolcallHistory = append(v.toolcallHistory, item)
}

func (v *Vars) ToolCalllog() (string, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	b, err := json.Marshal(v.toolcallHistory)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ... existing code ...
func (v *Vars) Clone() *Vars {
	clone := &Vars{
		// ChatID: v.ChatID,
		// New:        v.New,
		// MaxHistory: v.MaxHistory,
		// MaxSpan:    v.MaxSpan,
		// Context:    v.Context,
		//
		// MaxTurns: v.MaxTurns,
		// MaxTime:  v.MaxTime,
		// Models:   v.Models,
		//
		// Format: v.Format,
		//
		// Unsafe: v.Unsafe,
		// Workspace: v.Workspace,
		//
		// LogLevel: v.LogLevel,
		//

		// Extra:   make(map[string]string),
		history: make([]*Message, len(v.history)),
		Global:  v.Global.Clone(),
	}

	// Copy the History slice
	copy(clone.history, v.history)

	return clone
}

// Clear messages from history
func (v *Vars) ClearHistory() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.history = []*Message{}
}

func (v *Vars) SetHistory(messages []*Message) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.history = messages
}

// Append messages to history
func (v *Vars) AddHistory(messages []*Message) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.history = append(v.history, messages...)
}

// Return a copy of all current messages in history
func (v *Vars) ListHistory() []*Message {
	v.mu.RLock()
	defer v.mu.RUnlock()
	hist := make([]*Message, len(v.history))
	copy(hist, v.history)
	return hist
}

func NewVars() *Vars {
	return &Vars{
		Global: NewEnvironment(),
	}
}

// func (r *Vars) IsTrace() bool {
// 	return r.LogLevel == Tracing
// }

// func (r *Vars) Get(key string) (any, bool) {
// 	if r.Global == nil {
// 		return "", false
// 	}
// 	return r.Global.Get(key)
// }

// func (r *Vars) GetString(key string) string {
// 	if r.Global == nil {
// 		return ""
// 	}
// 	val, ok := r.Global.Get(key)
// 	if !ok {
// 		return ""
// 	}
// 	if s, ok := val.(string); ok {
// 		return s
// 	}
// 	return fmt.Sprintf("%v", val)
// }
