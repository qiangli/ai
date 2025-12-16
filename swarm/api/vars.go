package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"sync"

	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

type System = vos.System
type Workspace = vfs.Workspace

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

// Return value for key
func (g *Environment) Get(key string) (any, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if v, ok := g.env[key]; ok {
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

type ActionRTEnv struct {
	Base      string
	User      *User
	Roots     *Roots
	Workspace Workspace
	OS        System
	Secrets   SecretStore
}

// global context
type Vars struct {
	Global *Environment `json:"-"`

	// conversation history
	// history []*Message `json:"-"`

	// TODO persist to file
	// toolcallHistory []*ToolCallEntry `json:"-"`

	//
	// Middlewares []Middleware

	RootAgent *Agent

	RTE *ActionRTEnv

	mu sync.RWMutex
}

// fs.FS interface
func (v *Vars) Open(s string) (fs.File, error) {
	return v.RTE.Workspace.OpenFile(s, os.O_RDWR, 0o755)
}

// Return secret token for the current user
func (v *Vars) Token(key string) (string, error) {
	return v.RTE.Secrets.Get(v.RTE.User.Email, key)
}

// func (v *Vars) AddToolCall(item *ToolCallEntry) {
// 	v.mu.Lock()
// 	defer v.mu.Unlock()
// 	v.toolcallHistory = append(v.toolcallHistory, item)
// }

// func (v *Vars) ToolCalllog() (string, error) {
// 	v.mu.RLock()
// 	defer v.mu.RUnlock()
// 	b, err := json.Marshal(v.toolcallHistory)
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(b), nil
// }

func NewVars() *Vars {
	return &Vars{
		Global: NewEnvironment(),
	}
}

type ArgMap map[string]any

func NewArgMap() ArgMap {
	return make(map[string]any)
}

func (a ArgMap) Kitname() Kitname {
	kn := fmt.Sprintf("%s:%s", a.Kit(), a.Name())
	return Kitname(kn)
}

func (a ArgMap) Kit() string {
	return a.GetString("kit")
}

func (a ArgMap) Name() string {
	return a.GetString("name")
}

func (a ArgMap) Type() string {
	return a.GetString("type")
}

func (a ArgMap) Query() string {
	return a.GetString("query")
}

func (a ArgMap) SetQuery(query any) ArgMap {
	a["query"] = query
	return a
}

func (a ArgMap) Actions() []string {
	obj := a["actions"]
	if v, ok := obj.([]string); ok {
		return v
	}
	if v, ok := obj.(string); ok {
		var sa []string
		if err := json.Unmarshal([]byte(v), &sa); err == nil {
			return sa
		}
		return []string{v}
	}
	return []string{}
}
