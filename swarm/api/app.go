package api

// agent or tool
type AgentTool struct {
	Agent *Agent
	Tool  *ToolFunc
}

// per session values for AppConfig
type App struct {
	// root agent
	Agent *Agent

	// user
	User *User

	// chat id to continue the conersation
	// <config_base>/chat/<id>.json
	ChatID string

	// history @agent
	Context    string
	MaxHistory int
	MaxSpan    int

	MaxTime  int
	MaxTurns int

	Models string

	//
	LogLevel string

	Unsafe bool

	//
	Workspace string

	//
	Stdin  string
	Stdout string
	Stderr string

	//
	Config *AppConfig
}
