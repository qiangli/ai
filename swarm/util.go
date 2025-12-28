package swarm

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
)

// var essentialEnv = []string{"PATH", "PWD", "HOME", "USER", "SHELL", "GOPATH"}

// informative logging max value length
const maxInfoTextLen = 32

// ClearAllEnv clears all environment variables execep for the keeps
func ClearAllEnv(keeps []string) {
	var memo = make(map[string]bool, len(keeps))
	for _, key := range keeps {
		memo[key] = true
	}

	for _, env := range os.Environ() {
		key := strings.Split(env, "=")[0]
		if !memo[key] {
			os.Unsetenv(key)
		}
	}
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
	if input == nil {
		return nil, nil
	}
	if result, ok := input.(map[string]any); ok {
		return result, nil
	}

	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %v", err)
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map[string]any: %v", err)
	}

	return obj, nil
}

// func toPascalCase(name string) string {
// 	words := strings.FieldsFunc(name, func(r rune) bool {
// 		return r == '_' || r == '-'
// 	})
// 	tc := cases.Title(language.English)

// 	for i := range words {
// 		words[i] = tc.String(words[i])
// 	}
// 	return strings.Join(words, "")
// }

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

// return the first non empty string
func nvl(a ...string) string {
	for _, v := range a {
		if v != "" {
			return v
		}
	}
	return ""
}

// return the first non zero value
func nzl(a ...int) int {
	for _, v := range a {
		if v > 0 {
			return v
		}
	}
	return 0
}

// head trims the string to the maxLen and replaces newlines with /.
func head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "•")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func formatArgs(args map[string]any) string {
	var sb strings.Builder
	sb.WriteString("map[")
	for k, v := range args {
		if s, ok := v.(string); ok && len(s) > maxInfoTextLen {
			sb.WriteString(fmt.Sprintf("%s: %s [%v],", k, abbreviate(s, maxInfoTextLen), len(s)))
			continue
		}
		// sb.WriteString(fmt.Sprintf("%s: %t (type),", k, v))
	}
	sb.WriteString("]")
	return sb.String()
}

// abbreviate trims the string, keeping the beginning and end if exceeding maxLen.
// after replacing newlines with .
func abbreviate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "•")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)

	if len(s) > maxLen {
		// Calculate the length for each part
		keepLen := (maxLen - 3) / 2
		start := s[:keepLen]
		end := s[len(s)-keepLen:]
		return start + "..." + end
	}
	return s
}

func NilSafe[T any](ptr *T) T {
	var zeroValue T
	if ptr != nil {
		return *ptr
	}
	return zeroValue
}

// Return agent/kit name:
//   - agent: pack[/sub]
//   - tool: kit:name
//
// if the path matches the following pattern:
// agents/pack/agent.yaml
// agents/pack/pack.yaml
// agents/pack/sub.yaml
//
// tools/kit/name.yaml
func ActionNameFromFile(file string) string {
	parent := path.Dir(file)
	top := path.Base(path.Dir(parent))
	group := path.Base(parent)
	name := strings.TrimSuffix(path.Base(file), path.Ext(file))

	switch top {
	case "tools":
		return strings.ToLower(group + ":" + name)
	case "agents":
		if name == group || name == "agent" {
			return strings.ToLower(group)
		}
		return strings.ToLower(group + "/" + name)
	}
	// invalid
	return ""
}

func tail(data string, n int) string {
	lines := strings.Split(data, "\n")
	if n < len(lines) {
		// Return the lines after the nth line
		return strings.Join(lines[n:], "\n")
	}
	return ""
}

func count(obj any) int {
	switch v := obj.(type) {
	case []byte: // uint8
		return len(v)
	case string:
		return len(splitLines(v))
	case []int, []int8, []int16, []int32, []int64,
		[]uint, []uint16, []uint32, []uint64,
		[]string, []float64, []float32, []struct{}:
		return reflect.ValueOf(obj).Len()
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		complex64, complex128:
		// Return 1 for single value
		return 1
	case struct{}:
		return reflect.TypeOf(obj).NumField()
	default:
		return 0
	}
}

func splitLines(text string) []string {
	return strings.Split(text, "\n")
}
