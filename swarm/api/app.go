package api

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
