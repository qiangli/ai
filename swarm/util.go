package swarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"text/template"

	"github.com/kaptinlin/jsonrepair"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}

	ip := net.ParseIP(host)

	if ip != nil && ip.IsLoopback() {
		return true
	}

	if host == "localhost" {
		return true
	}

	return false
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

func expandWithDefault(input string) string {
	return os.Expand(input, func(key string) string {
		parts := strings.SplitN(key, ":-", 2)
		value := os.Getenv(parts[0])
		if value == "" && len(parts) > 1 {
			return parts[1]
		}
		return value
	})
}

// func toModelLevel(s string) model.Level {
// 	switch s {
// 	case "L0":
// 		return model.L0
// 	case "L1":
// 		return model.L1
// 	case "L2":
// 		return model.L2
// 	case "L3":
// 		return model.L3
// 	case "Image":
// 		return model.Image
// 	}
// 	return model.L0
// }

func toPascalCase(name string) string {
	words := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	tc := cases.Title(language.English)

	for i := range words {
		words[i] = tc.String(words[i])
	}
	return strings.Join(words, "")
}

// tryUnmarshal tries to unmarshal the data into the v.
// If it fails, it will try to repair the data and unmarshal it again.
func tryUnmarshal(data string, v any) error {
	err := json.Unmarshal([]byte(data), v)
	if err == nil {
		return nil
	}

	repaired, err := jsonrepair.JSONRepair(data)
	if err != nil {
		return fmt.Errorf("failed to repair JSON: %v", err)
	}
	return json.Unmarshal([]byte(repaired), v)
}

// baseCommand returns the last part of the string separated by /.
func baseCommand(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	sa := strings.Split(s, "/")
	return sa[len(sa)-1]
}
