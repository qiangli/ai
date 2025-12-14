package resource

import (
	"embed"
)

//go:embed standard/*
var ResourceFS embed.FS

func NewStandardStore() *ResourceStore {
	return &ResourceStore{
		Base: "standard",
		FS:   ResourceFS,
	}
}

//go:embed shell_security_system.md
var ShellSecuritySystemRole string

//go:embed shell_security_user.md
var ShellSecurityUserRole string

//go:embed root/agent.yaml
var RootAgentData []byte

//go:embed root/format.json.txt
var formatJson string

//go:embed root/format.markdown.txt
var formatMarkdown string

//go:embed root/format.text.txt
var formatText string

func FormatFile(format string) string {
	switch format {
	case "json":
		return formatJson
	case "markdown", "md":
		return formatMarkdown
	case "text", "txt":
		return formatText
	default:
		return ""
	}
}
