{{- /*PR description output format*/ -}}
# PR Description

## Title

{{.Title}}

## Type

{{range .Types -}}
- [x] {{ . }}
{{end}}

## Description

{{.Description}}

## Screenshots

## Changes walkthrough
{{- range .Files }}
### {{.Filename}}
- **Label**: {{.Label}}
- **Change Title**: {{.Title}}
- **Change Summary**:
{{- with .Summary }}
{{- range $line := splitLines . }}
  {{ $line }}
{{- end }}
{{- end }}
{{ end }}

## Related Issues
