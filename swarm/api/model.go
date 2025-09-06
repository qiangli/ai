package api

type ModelsConfig struct {
	Owner string `yaml:"owner"`

	Alias string `yaml:"alias" json:"alias"`

	// // model
	// Model string `yaml:"model" json:"model"`

	// default for Models
	Provider string `yaml:"provider" json:"provider"`
	BaseUrl  string `yaml:"base_url" json:"baseUrl"`
	ApiKey   string `yaml:"api_key" json:"apiKey"`

	Models map[string]*ModelConfig `yaml:"models" json:"models"`
}

type ModelConfig struct {
	// key of models map in models config
	// Name string `yaml:"name"`
	// Features map[Feature]bool `yaml:"features" json:"features"`

	// //
	// Type        string `yaml:"type"`
	// Type OutputType `yaml:"type" json:"type"`

	// Description string `yaml:"description"`

	// Provider    string `yaml:"provider"`
	// Model       string `yaml:"model"`
	// BaseUrl     string `yaml:"baseUrl"`
	// ApiKey      string `yaml:"apiKey"`

	Provider string `yaml:"provider" json:"provider"`

	Model   string `yaml:"model" json:"model"`
	BaseUrl string `yaml:"base_url" json:"baseUrl"`
	ApiKey  string `yaml:"api_key" json:"apiKey"`
}
