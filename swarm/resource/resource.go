package resource

import (
	"embed"
)

//go:embed core/*
var ResourceFS embed.FS

func NewCoreStore() *ResourceStore {
	return &ResourceStore{
		Base: "core",
		FS:   ResourceFS,
	}
}

// //go:embed shell_security_system.md
// var ShellSecuritySystemRole string

// //go:embed shell_security_user.md
// var ShellSecurityUserRole string

//go:embed core/agents/root/agent.yaml
var RootAgentData []byte

//go:embed core/agents/root/format.json.txt
var formatJson string

//go:embed core/agents/root/format.markdown.txt
var formatMarkdown string

//go:embed core/agents/root/format.text.txt
var formatText string

func FormatFile(format string) string {
	switch format {
	case "json":
		return formatJson
	case "markdown", "md":
		return formatMarkdown
	case "text", "txt", "raw":
		return formatText
	default:
		return ""
	}
}
