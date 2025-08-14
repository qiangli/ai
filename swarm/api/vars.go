package api

import (
	"fmt"
	// "strings"

	"github.com/qiangli/ai/swarm/api/model"
)

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

// global context
type Vars struct {
	Config *AppConfig `json:"config"`

	OS        string            `json:"os"`
	Arch      string            `json:"arch"`
	ShellInfo map[string]string `json:"shell_info"`
	OSInfo    map[string]string `json:"os_info"`

	UserInfo map[string]string `json:"user_info"`

	UserInput *UserInput `json:"user_input"`

	Workspace string `json:"workspace"`
	// Repo      string `json:"repo"`
	Home string `json:"home"`
	Temp string `json:"temp"`

	// EnvType indicates the environment type where the agent is running
	// It can be "container" for Docker containers or "host" for the host machine
	EnvType string `json:"env_type"`

	Roots []string `json:"roots"`

	// per agent
	Extra map[string]any `json:"extra"`

	Models map[model.Level]*model.Model `json:"models"`

	//
	ToolRegistry map[string]*ToolFunc `json:"tool_registry"`
	// AgentRegistry map[string]*AgentsConfig `json:"agent_registry"`

	// agent -> Resources
	// ResourceMap map[string]*Resource

	AdviceMap       map[string]Advice
	EntrypointMap   map[string]Entrypoint
	TemplateFuncMap TemplateFuncMap

	// conversation history
	History []*Message
}

// func (r *Vars) Resource(agent, name string) ([]byte, error) {
// 	key := strings.SplitN(agent, "/", 2)[0]
// 	res, ok := r.ResourceMap[key]
// 	if !ok {
// 		return nil, fmt.Errorf("no resource found for %q", agent)
// 	}
// 	b, err := res.Content(name)
// 	if err != nil {
// 		return nil, fmt.Errorf("error loading %s for %s: %w", name, agent, err)
// 	}
// 	return b, nil
// }

type Resource struct {
	ID string `json:"id"`

	// key: scheme:path.type
	Content func(string) ([]byte, error) `json:"content"`
}

func (r *Vars) ListTools() []*ToolFunc {
	tools := make([]*ToolFunc, 0, len(r.ToolRegistry))
	for _, tool := range r.ToolRegistry {
		tools = append(tools, tool)
	}
	return tools
}

// func (r *Vars) ListAgents() map[string]*AgentConfig {
// 	agents := make(map[string]*AgentConfig)
// 	for _, v := range r.AgentRegistry {
// 		for _, agent := range v.Agents {
// 			if v.Internal && !r.Config.Internal {
// 				continue
// 			}
// 			agents[agent.Name] = agent
// 		}
// 	}
// 	return agents
// }

func NewVars() *Vars {
	return &Vars{
		Extra: map[string]any{},
	}
}

func (r *Vars) Get(key string) any {
	if r.Extra == nil {
		return nil
	}
	return r.Extra[key]
}

func (r *Vars) GetString(key string) string {
	if r.Extra == nil {
		return ""
	}
	v, ok := r.Extra[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}
