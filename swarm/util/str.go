package util

import (
	"strings"
)

// change name to lower case and replace all space with dash "-"
func NormalizedName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.ReplaceAll(name, " ", "-")
}

// Clip trims the string to the maxLen
func Clip(text string, maxLen int) string {
	if len(text) > maxLen {
		return text[:maxLen] + "..."
	}
	return text
}

// Head trims the string to the maxLen and replaces newlines with /.
func Head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "/")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
