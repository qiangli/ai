package api

type Action struct {
	Name      string
	Arguments map[string]any

	//
	Tool  *ToolFunc
	Agent *Agent
}

// //
// type AppInput struct {
// 	Owner string

// 	// @ (LLM) or /
// 	// --agent
// 	// @<name> anonymous
// 	Name string

// 	// agent
// 	// --message + args[x:] where x is remaining non flag/option args
// 	Message     string
// 	Instruction string

// 	// --tools
// 	Functions string

// 	Models string

// 	MaxTurns int
// 	MaxTime  int

// 	MaxHistory int
// 	MaxSpan    int

// 	Format   string
// 	LogLevel string

// 	// all args including name
// 	// command line: args
// 	// json
// 	// Arguments map[string]any
// }

// agent or tool
// TODO redesign: agent <-> tool same?
type AgentTool struct {
	Agent *Agent
	Tool  *ToolFunc

	//
	Owner       string
	Instruction string
	Message     string
	Arguments   map[string]any
}
