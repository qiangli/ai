package swarm

import (
	"fmt"

	"github.com/qiangli/ai/internal/llm"
)

type AgentRegistry map[string]*Agent

var Agents = AgentRegistry{}

// type Message struct {
// 	Role    string
// 	Content string

// 	Sender string
// }

type Message = llm.Message

// type Request = llm.Request

type ToolFunc = llm.ToolFunc

// type ToolCall = llm.ToolCall
type ToolCall = llm.ToolCall

const (
	RoleSystem    = "system"
	RoleAssistant = "assistant"
	RoleUser      = "user"
	RoleTool      = "tool"
)

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

type Vars struct {
	OS        string
	Arch      string
	ShellInfo map[string]string
	OSInfo    map[string]string

	UserInfo map[string]string

	Workspace string
	WorkDir   string

	Agent      string `json:"agent"`
	Subcommand string `json:"subcommand"`
	Input      string
	Intent     string
	Query      string
	Files      []string `json:"files"`

	Env string

	// per agent
	Extra map[string]any
}

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

// type InputText *string

// type OutputText *string

// map[string]interface{}

// type Input struct {
// 	Name      string         `json:"name"`
// 	Arguments map[string]any `json:"arguments"`

// 	Vars Vars `json:"vars"`
// }

// type Output struct {
// 	Output     string `json:"output"`
// 	Error      string `json:"error"`
// 	ExitStatus int    `json:"exit_status"`

// 	Agent *Agent `json:"agent"`
// }

// Swarm Agents can call functions directly.
// Function should usually return a string (values will be attempted to be cast as a string).
// If a function returns an Agent, execution will be transferred to that Agent.
// type Function func(*Input) *Output

// type Function struct {
// 	Name        string
// 	Description string
// 	Parameters  map[string]any
// }

// // Agent instructions are directly converted into the system prompt of a conversation.
// // The instructions can either be a regular string, or a function that returns a string.
// type Instruction struct {
// 	Content string
// 	Func    func(Vars) (string, error)
// }

// func (r Instruction) Get(vars Vars) (string, error) {
// 	if r.Func != nil {
// 		return r.Func(vars)
// 	}
// 	return r.Content, nil
// }

type Agent struct {
	// The name of the agent.
	Name string

	Display string

	// The model to be used by the agent
	Model *Model

	// The role of the agent. default is "system"
	Role string

	// Instructions for the agent, can be a string or a callable returning a string
	Instruction string

	// A list of functions that the agent can call
	Functions []*ToolFunc

	// advices

	BeforeAdvice Advice
	AfterAdvice  Advice

	AroundAdvice Advice
}

type Request struct {
	Message *Message
	Vars    *Vars
}

type Response struct {
	// A list of message objects generated during the conversation
	// with a sender field indicating which Agent the message originated from.
	Messages []*Message

	// The last agent to handle a message
	Agent *Agent

	// The same as the input variables, plus any changes.
	Vars *Vars
}

func (r *Response) LastMessage() *Message {
	if len(r.Messages) > 0 {
		return r.Messages[len(r.Messages)-1]
	}
	return nil
}

func (r *Response) AddExtra(key string, value any) {
	if r.Vars == nil {
		r.Vars = &Vars{}
	}
	if r.Vars.Extra == nil {
		r.Vars.Extra = map[string]interface{}{}
	}
	r.Vars.Extra[key] = value
}

// Result encapsulates the possible return values for an agent function.
type Result struct {
	// The result value as a string
	Value string

	// The agent instance, if applicable
	Agent *Agent

	// A dictionary of context variables
	Vars Vars
}

type Model struct {
	Name string
	// Provider string
	BaseUrl string
	ApiKey  string
}

// type Text *string

// type Agentic interface {
// 	Name() string
// 	Instruction() string
// 	Input() string
// 	Model() Model
// 	ToolCalls() []ToolCall
// }

// type Descriptor interface {
// 	Help() string
// 	Describe() string
// 	Version() string
// }

// type Commander interface {
// 	Execute(Text) Text
// }

// type Treelike interface {
// 	Sub() []Agentic
// }

// type ToolDefinition struct {
// 	Name        string         `json:"name"`
// 	Description string         `json:"description"`
// 	Parameters  ToolParameters `json:"parameters"`
// }

// type ToolParameters struct {
// 	Type       string         `json:"type"`
// 	Properties map[string]any `json:"properties"`
// 	Required   []string       `json:"required"`
// }

// type Function func(ctx context.Context, name string, args map[string]any) (string, error)

type Advice func(*Request, *Response, Advice) error
