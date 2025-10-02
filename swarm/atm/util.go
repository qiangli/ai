package atm

import (
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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

// baseCommand returns the last part of the string separated by /.
func baseCommand(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	sa := strings.Split(s, "/")
	return sa[len(sa)-1]
}

// split2 splits string s by sep into two parts. if there is only one part,
// use val as the second part
func split2(s string, sep string, val string) (string, string) {
	var p1, p2 string
	parts := strings.SplitN(s, sep, 2)
	if len(parts) == 1 {
		p1 = parts[0]
		p2 = val
	} else {
		p1 = parts[0]
		p2 = parts[1]
	}
	return p1, p2
}

func nvl(sa ...string) string {
	for _, v := range sa {
		if v != "" {
			return v
		}
	}
	return ""
}

// head trims the string to the maxLen and replaces newlines with /.
func head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "/")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
