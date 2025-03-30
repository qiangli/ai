package api

import (
	"html/template"
)

type TemplateFuncMap = template.FuncMap

type UserConfig struct {
	Name    string `yaml:"name"`
	Display string `yaml:"display"`
}

type AgentsConfig struct {
	User      UserConfig       `yaml:"user"`
	Agents    []AgentConfig    `yaml:"agents"`
	Functions []FunctionConfig `yaml:"functions"`
	Models    []ModelConfig    `yaml:"models"`

	MaxTurns int `yaml:"maxTurns"`
	MaxTime  int `yaml:"maxTime"`

	ResourceMap     map[string]string     `yaml:"-"`
	AdviceMap       map[string]Advice     `yaml:"-"`
	EntrypointMap   map[string]Entrypoint `yaml:"-"`
	TemplateFuncMap TemplateFuncMap       `yaml:"-"`
}

type AgentConfig struct {
	Name        string `yaml:"name"`
	Display     string `yaml:"display"`
	Description string `yaml:"description"`

	//
	Instruction PromptConfig `yaml:"instruction"`

	Model string `yaml:"model"`

	Entrypoint string `yaml:"entrypoint"`

	Functions []string `yaml:"functions"`

	Dependencies []string `yaml:"dependencies"`

	Advices AdviceConfig `yaml:"advices"`
}

type PromptConfig struct {
	Role    string `yaml:"role"`
	Content string `yaml:"content"`
}

type FunctionConfig struct {
	Label   string `yaml:"label"`
	Service string `yaml:"service"`

	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`
}

type ModelConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Model       string `yaml:"model"`
	BaseUrl     string `yaml:"baseUrl"`
	ApiKey      string `yaml:"apiKey"`
	External    bool   `yaml:"external"`
}

type AdviceConfig struct {
	Before string `yaml:"before"`
	After  string `yaml:"after"`
	Around string `yaml:"around"`
}

// type Agent struct {
// 	// The name of the agent.
// 	Name string

// 	Display string

// 	// The model to be used by the agent
// 	Model *Model

// 	// The role of the agent. default is "system"
// 	Role string

// 	// Instructions for the agent, can be a string or a callable returning a string
// 	Instruction string

// 	RawInput *UserInput

// 	// Vars *Vars

// 	// Functions that the agent can call
// 	Tools []*ToolFunc

// 	Entrypoint Entrypoint

// 	Dependencies []*Agent

// 	// advices
// 	BeforeAdvice Advice
// 	AfterAdvice  Advice
// 	AroundAdvice Advice

// 	//
// 	MaxTurns int
// 	MaxTime  int

// 	//
// 	// sw *Swarm
// }
