package atm

import (
	"bytes"
	"strings"
	"text/template"
)

func maxLen(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

var tplFuncMap = template.FuncMap{
	"maxLen":     maxLen,
	"trim":       strings.TrimSpace,
	"toLower":    strings.ToLower,
	"splitLines": splitLines,
}

func applyTemplate(tpl string, data any, funcMap template.FuncMap) (string, error) {
	t, err := template.New("swarm").Funcs(funcMap).Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
