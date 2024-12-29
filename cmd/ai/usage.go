package main

const rootUsageTemplate = `Usage:
  ai [OPTIONS] COMMAND [message...]{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Commands:
  /[binary] [message...]  Help with command and shell scripting
  @[agent]  [message...]  Assistants for a variety of tasks

Available Binary:
  list                    List available binaries on the path

Supported Agent:
  ask                     Ask general questions
{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}

Environment variables:
  AI_API_KEY, AI_BASE_URL, AI_CONFIG, AI_DEBUG, AI_DRY_RUN, AI_DRY_RUN_FILE, AI_EDITOR, AI_MODEL

Use "{{.CommandPath}} help" for more info.
`
