package api

type McpServerConfig struct {
	Transport string `json:"transport"`

	ServerUrl string `json:"serverUrl"`

	Command string         `json:"command"`
	Args    []string       `json:"args"`
	Env     map[string]any `json:"env"`

	Server string `json:"-"`
}
