package util

// import (
// 	"encoding/json"
// 	"strings"
// 	"text/template"
// 	"time"
// )

// func maxLen(s string, max int) string {
// 	if len(s) > max {
// 		return s[:max] + "..."
// 	}
// 	return s
// }

// func splitLines(s string) []string {
// 	return strings.Split(s, "\n")
// }

// // TODO: jq, *Case
// // https://masterminds.github.io/sprig/
// var TplFuncMap = template.FuncMap{
// 	"maxLen":     maxLen,
// 	// "trim":       strings.TrimSpace,
// 	// "toLower":    strings.ToLower,
// 	// "toUpper":    strings.ToUpper,
// 	// "splitLines": splitLines,
// 	// "prettyJson": func(s string) string {
// 	// 	v, err := prettyJson(s)
// 	// 	if err != nil {
// 	// 		return err.Error()
// 	// 	}
// 	// 	return v
// 	// },
// 	// "now": now,
// }

// func prettyJson(obj any) (string, error) {
// 	jsonData, err := json.MarshalIndent(obj, "", "  ")
// 	if err != nil {
// 		return "", err
// 	}
// 	return string(jsonData), nil
// }

// func now() string {
// 	return time.Now().Format(time.RFC3339)
// }
