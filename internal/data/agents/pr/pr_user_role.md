{{- /*PR user prompt - description, review, code suggestion*/ -}}
{{- if .instruction}}
Additional instructions:

{{.instruction | trim}}

======
{{- end}}

The PR Git Diff:

======
{{ .diff | trim }}
======

{{if .changelog }}

Current date:

```text
{{ .today }}
```

The current 'CHANGELOG.md' file

======
{{ .changelog }}
======

{{end}}
