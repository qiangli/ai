package swarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

func applyTemplate(tpl string, data any, funcMap template.FuncMap) (string, error) {
	t, err := template.New("config").Funcs(funcMap).Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func clip(s string, max int) string {
	if max > 0 && len(s) > max {
		trailing := "..."
		if s[len(s)-1] == '\n' || s[len(s)-1] == '\r' {
			trailing = "...\n"
		}
		s = s[:max] + trailing
	}
	return s
}

func structToMap(input any) (map[string]any, error) {
	if result, ok := input.(map[string]any); ok {
		return result, nil
	}

	jsonData, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input to JSON: %v", err)
	}

	var resultMap map[string]any
	if err := json.Unmarshal(jsonData, &resultMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to map[string]any: %v", err)
	}

	return resultMap, nil
}
