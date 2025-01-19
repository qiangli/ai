package resource

import (
	"regexp"
	"strings"
	"text/template"
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
