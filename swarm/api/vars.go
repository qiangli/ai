package api

import (
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

// global context
type Vars struct {
	Global *Environment `json:"global"`

	RootAgent *Agent `json:"root_agent"`

	Base string

	SessionID SessionID
	User      *User

	Roots     *Roots
	Workspace Workspace
	OS        System
	Secrets   SecretStore
	Assets    AssetManager
	Blobs     BlobStore
	Tools     ToolSystem
	Adapters  AdapterRegistry
	History   MemStore
	Log       CallLogger
}

// Return default query from message and content.
// Return error if either message or content is a template
// Preprocessing is required for templates using [ai:build_query]
func (r *Vars) DefaultQuery(argm ArgMap) (string, error) {
	message := argm.GetString("message")
	if IsTemplate(message) {
		return "", fmt.Errorf("message is a template")
	}
	if message != "" {
		v, err := r.loadContent(message)
		if err != nil {
			return "", err
		}
		message = v
	}

	return message, nil
}

// Return default prompt using instruction.
// Return error if instruction is a template
// Return empty string if no instruction if found
// Preprocessing is required for templates using [ai:build_prompt]
func (r *Vars) DefaultPrompt(argm ArgMap) (string, error) {
	instruction := argm.GetString("instruction")
	if instruction == "" {
		return "", nil
	}
	if IsTemplate(instruction) {
		return "", fmt.Errorf("instruction is a template")
	}
	prompt, err := r.loadContent(instruction)
	if err != nil {
		return "", err
	}
	return prompt, nil
}

func (r *Vars) LoadScript(s string) (string, error) {
	return LoadURIContent(r.Workspace, s)
}

// return as is for non URI
func (r *Vars) loadContent(s string) (string, error) {
	if !IsURI(s) {
		return s, nil
	}
	return LoadURIContent(r.Workspace, s)
}

// fs.FS interface
func (r *Vars) Open(s string) (fs.File, error) {
	return r.Workspace.OpenFile(s, os.O_RDWR, 0o755)
}

// Return secret token for the current user
func (r *Vars) Token(key string) (string, error) {
	return r.Secrets.Get(r.User.Email, key)
}

// func NewVars() *Vars {
// 	return &Vars{
// 		Global: NewEnvironment(),
// 	}
// }

type ArgMap map[string]any

func NewArgMap() ArgMap {
	return make(map[string]any)
}

func (a ArgMap) Arg(key string) any {
	return a.Get(key)
}

func (a ArgMap) SetArg(key string, value any) ArgMap {
	a[key] = value
	return a
}

func (a ArgMap) Kitname() Kitname {
	kit := a.GetString("kit")
	name := a.GetString("name")
	if kit == "agent" {
		pack := a.GetString("pack")
		if pack == "" {
			pack = "missing"
		}
		name = pack + "/" + name
	}
	kn := fmt.Sprintf("%s:%s", kit, name)
	return Kitname(kn)
}

func (a ArgMap) Type() string {
	return a.GetString("type")
}

func (a ArgMap) Message() string {
	return a.GetString("message")
}

func (a ArgMap) HasQuery() bool {
	_, ok := a["query"]
	return ok
}

func (a ArgMap) Query() string {
	return a.GetString("query")
}

func (a ArgMap) SetQuery(query any) ArgMap {
	a["query"] = query
	return a
}

func (a ArgMap) HasPrompt() bool {
	_, ok := a["prompt"]
	return ok
}

func (a ArgMap) Prompt() string {
	return a.GetString("prompt")
}

func (a ArgMap) SetPrompt(prompt any) ArgMap {
	a["prompt"] = prompt
	return a
}

func (a ArgMap) Actions() []string {
	obj := a["actions"]
	return ToStringArray(obj)
}

// check and return an instance of agent.
func (a ArgMap) Agent() *Agent {
	v, found := a["agent"]
	if !found {
		return nil
	}
	if agent, ok := v.(*Agent); ok {
		return agent
	}
	return nil
}

func (a ArgMap) Action() *Action {
	v, found := a["action"]
	if !found {
		return nil
	}
	if action, ok := v.(*Action); ok {
		return action
	}
	if s, ok := v.(string); ok {
		return &Action{
			Command: s,
		}
	}
	return nil
}

func (a ArgMap) HasHistory() bool {
	_, ok := a["history"]
	return ok
}

func (a ArgMap) History() []*Message {
	v, found := a["history"]
	if !found {
		return nil
	}
	if history, ok := v.([]*Message); ok {
		return history
	}
	return nil
}

func (a ArgMap) SetHistory(messages []*Message) ArgMap {
	a["history"] = messages
	return a
}

func (a ArgMap) DeleteHitory() {
	delete(a, "history")
}
