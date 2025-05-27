package mcp

import (
	"os"
	"strings"
)

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
