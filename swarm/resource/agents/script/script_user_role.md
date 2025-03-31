{{- if .Command -}}
Please assist with the **{{ .Command }}** command.

- If direct execution of the command is appropriate, the output of the command should be provided as well as the command used.
- If executing the command is not appropriate or possible, suggest a complete command with concise explanations.
- If appropriate, the command may be executed via available tools `exec` to provide accurate results.

Refer to the request below:
{{- end}}
{{ .Message }}
