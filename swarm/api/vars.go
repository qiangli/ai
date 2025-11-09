package api

import (
	"fmt"
	"maps"
	"sync"
)

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

type Global struct {
	env map[string]any
	mu  sync.RWMutex
}

func (g *Global) Get(key string) (any, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if v, ok := g.env[key]; ok {
		return v, ok
	}
	return nil, false
}

func (g *Global) Set(key string, val any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.env[key] = val
}

// copy all src values to the global env
func (g *Global) Add(src map[string]any) {
	g.mu.Lock()
	defer g.mu.Unlock()
	maps.Copy(g.env, src)
}

// thread safe access to the env
func (g *Global) Apply(fn func(map[string]any) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return fn(g.env)
}

// copy all global env to dst
func (g *Global) Copy(dst map[string]any) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	maps.Copy(dst, g.env)
}

func (g *Global) Clone() *Global {
	g.mu.Lock()
	defer g.mu.Unlock()

	env := make(map[string]any)
	maps.Copy(env, g.env)
	return &Global{
		env: env,
	}
}

func NewGlobal() *Global {
	return &Global{
		env: make(map[string]any),
	}
}

// global context
type Vars struct {
	LogLevel LogLevel `json:"log_level"`

	ChatID string `json:"chat_id"`
	// MaxTurns   int    `json:"max_turns"`
	// MaxTime    int    `json:"max_time"`
	// New        *bool  `json:"new"`
	// MaxHistory int    `json:"max_history"`
	// MaxSpan    int    `json:"max_span"`
	// Context    string `json:"context"`
	// Format     string `json:"format"`
	// Models     string `json:"models"`

	Unsafe bool `json:"unsafe"`
	// Workspace string `json:"workspace"`

	// DryRun        bool   `json:"-"`
	// DryRunContent string `json:"-"`

	// conversation history
	History []*Message `json:"-"`

	Global *Global `json:"-"`
}

func (v *Vars) Clone() *Vars {
	clone := &Vars{
		// ChatID:     v.ChatID,
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
		Unsafe: v.Unsafe,
		// Workspace: v.Workspace,
		//
		LogLevel: v.LogLevel,
		//
		// DryRun:        v.DryRun,
		// DryRunContent: v.DryRunContent,
		//
		// Extra:   make(map[string]string),
		History: make([]*Message, len(v.History)),
		Global:  v.Global.Clone(),
	}

	// Copy the History slice
	copy(clone.History, v.History)

	return clone
}

func NewVars() *Vars {
	return &Vars{
		Global: NewGlobal(),
	}
}

func (r *Vars) IsTrace() bool {
	return r.LogLevel == Tracing
}

func (r *Vars) Get(key string) (any, bool) {
	if r.Global == nil {
		return "", false
	}
	return r.Global.Get(key)
}

func (r *Vars) GetString(key string) string {
	if r.Global == nil {
		return ""
	}
	val, ok := r.Global.Get(key)
	if !ok {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}
