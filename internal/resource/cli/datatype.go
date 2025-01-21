package cli

type ConfigSchema struct {
	Config       string     `json:"config"` // required field
	Workspace    string     `json:"workspace,omitempty"`
	APIKey       string     `json:"api-key,omitempty"`
	Model        string     `json:"model,omitempty"`
	BaseURL      string     `json:"base-url,omitempty"`
	L1APIKey     string     `json:"l1-api-key,omitempty"`
	L1Model      string     `json:"l1-model,omitempty"`
	L1BaseURL    string     `json:"l1-base-url,omitempty"`
	L2APIKey     string     `json:"l2-api-key,omitempty"`
	L2Model      string     `json:"l2-model,omitempty"`
	L2BaseURL    string     `json:"l2-base-url,omitempty"`
	L3APIKey     string     `json:"l3-api-key,omitempty"`
	L3Model      string     `json:"l3-model,omitempty"`
	L3BaseURL    string     `json:"l3-base-url,omitempty"`
	Verbose      bool       `json:"verbose,omitempty"`
	Quiet        bool       `json:"quiet,omitempty"`
	Editor       string     `json:"editor,omitempty"`
	Role         string     `json:"role,omitempty"`
	RolePrompt   string     `json:"role-prompt,omitempty"`
	NoMetaPrompt bool       `json:"no-meta-prompt,omitempty"`
	Interactive  bool       `json:"interactive,omitempty"`
	PbRead       bool       `json:"pb-read,omitempty"`
	PbWrite      bool       `json:"pb-write,omitempty"`
	Log          string     `json:"log,omitempty"`
	Trace        bool       `json:"trace,omitempty"`
	Output       string     `json:"output,omitempty"`
	SQLConfig    *SqlConfig `json:"sqlConfig,omitempty"`
	GitConfig    *GitConfig `json:"gitConfig,omitempty"` // Assuming GitConfig is also a struct
}

type SqlConfig struct {
	DBHost     string `json:"db-host,omitempty"`
	DBPort     string `json:"db-port,omitempty"`
	DBUsername string `json:"db-username,omitempty"`
	DBPassword string `json:"db-password,omitempty"`
	DBName     string `json:"db-name,omitempty"`
}

type GitConfig struct {
}
