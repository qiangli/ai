package api

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/qiangli/shell/vfs"
	"github.com/qiangli/shell/vos"
)

type System = vos.System
type Workspace = vfs.Workspace

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

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
		// TODO: append note the message to indicate this is template?
		return message, nil
		// return "", fmt.Errorf("message is a template")
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
