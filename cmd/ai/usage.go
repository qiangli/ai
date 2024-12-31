package main

const rootUsageTemplate = `Usage:
  ai [OPTIONS] COMMAND [message...]{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Commands:
  /[binary] [message...]  Help with commands and shell scripting
  @[agent]  [message...]  Consult with agents for a variety of tasks

Miscellaneous:
  list                    List available binaries on the path
  info                    Show system information

Supported Agent:
  ask                     Ask general questions
{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Environment variables:
  AI_API_KEY, AI_BASE_URL, AI_CONFIG, AI_DEBUG, AI_DRY_RUN, AI_DRY_RUN_CONTENT, AI_EDITOR, AI_MODEL, AI_ROLE, AI_ROLE_CONTENT

Use "{{.CommandPath}} help" for more info.
`
