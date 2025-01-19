package resource

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/kaptinlin/jsonrepair"
)

func splitVersion(s string) (string, string) {
	// postgres version string format: PostgreSQL 15.8 on ...
	re := regexp.MustCompile(`(\w+) (\d+\.\d+(\.\d+)?)`)
	match := re.FindStringSubmatch(s)

	if len(match) > 2 {
		dialect := match[1]
		version := match[2]
		return dialect, version
	}
	return "", ""
}

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

func apply(tpl string, data any) (string, error) {
	t, err := template.New("pr").Funcs(tplFuncMap).Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
