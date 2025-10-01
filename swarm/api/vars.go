package api

const (
	VarsEnvContainer = "container"
	VarsEnvHost      = "host"
)

// global context
type Vars struct {
	Config *AppConfig `json:"config"`

	// OS        string            `json:"os"`
	// Arch      string            `json:"arch"`
	// ShellInfo map[string]string `json:"shell_info"`
	// OSInfo    map[string]string `json:"os_info"`

	// UserInfo map[string]string `json:"user_info"`

	// UserInput *UserInput `json:"user_input"`

	Workspace string `json:"workspace"`
	// // Repo      string `json:"repo"`
	// Home string `json:"home"`
	// Temp string `json:"temp"`

	// EnvType indicates the environment type where the agent is running
	// It can be "container" for Docker containers or "host" for the host machine
	// EnvType string `json:"env_type"`

	// Roots []string `json:"roots"`

	//
	Extra map[string]string `json:"extra"`

	// conversation history
	History []*Message
}

func NewVars() *Vars {
	return &Vars{
		Extra: map[string]string{},
	}
}

func (r *Vars) Get(key string) string {
	if r.Extra == nil {
		return ""
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
	return v
}
