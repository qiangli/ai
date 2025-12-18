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

func (g *Environment) Clone() *Environment {
	g.mu.Lock()
	defer g.mu.Unlock()

	env := make(map[string]any)
	maps.Copy(env, g.Env)
	return &Environment{
		Env: env,
	}
}

type ActionRTEnv struct {
	Base string

	//
	ID        string
	User      *User
	Roots     *Roots
	Workspace Workspace
	OS        System
	Secrets   SecretStore
	Assets    AssetManager
	Blobs     BlobStore

	// Tools ToolSystem
	// Adapters AdapterRegistry
}

// Return default query from message and content.
// Return error if either message or content is a template
func (r *ActionRTEnv) DefaultQuery(argm ArgMap) (string, error) {
	message := argm.GetString("message")
	if IsTemplate(message) {
		return "", fmt.Errorf("message is a template")
	}
	content := argm.GetString("content")
	if IsTemplate(content) {
		return "", fmt.Errorf("content is a template")
	}
	if content != "" {
		v, err := r.loadContent(content)
		if err != nil {
			return "", err
		}
		content = v
	}
	query := Cat(message, content, "\n###\n")
	return query, nil
}

// Return default prompt read from the instruction.
// Return error if instruction is a template
// Return empty string if no instruction if found
func (r *ActionRTEnv) DefaultPrompt(argm ArgMap) (string, error) {
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

func (r *ActionRTEnv) LoadScript(v string) (string, error) {
	return LoadURIContent(r.Workspace, v)
}

// return as is for non URI
func (r *ActionRTEnv) loadContent(v string) (string, error) {
	if !IsURI(v) {
		return v, nil
	}
	return LoadURIContent(r.Workspace, v)
}

// global context
type Vars struct {
	Global *Environment `json:"global"`

	RootAgent *Agent `json:"root_agent"`

	RTE *ActionRTEnv `json:"-"`
	mu  sync.RWMutex `json:"-"`
}

// fs.FS interface
func (v *Vars) Open(s string) (fs.File, error) {
	return v.RTE.Workspace.OpenFile(s, os.O_RDWR, 0o755)
}

// Return secret token for the current user
func (v *Vars) Token(key string) (string, error) {
	return v.RTE.Secrets.Get(v.RTE.User.Email, key)
}

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
