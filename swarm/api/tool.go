package api

type ToolConfig struct {
	Kit string `yaml:"kit"`

	Internal bool `yaml:"internal"`

	Type        string         `yaml:"type"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`

	Body string `yaml:"body"`
}

type ToolsConfig struct {
	Kit string `yaml:"kit"`

	Internal bool `yaml:"internal"`

	// system commands used by tools
	Commands []string `yaml:"commands"`

	Tools []*ToolConfig `yaml:"tools"`
}
