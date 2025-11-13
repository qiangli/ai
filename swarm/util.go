package swarm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/u-root/u-root/pkg/shlex"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
)

var essentialEnv = []string{"PATH", "PWD", "HOME", "USER", "SHELL"}

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

// // baseCommand returns the last part of the string separated by /.
// func baseCommand(s string) string {
// 	s = strings.TrimSpace(s)
// 	s = strings.Trim(s, "/")
// 	sa := strings.Split(s, "/")
// 	return sa[len(sa)-1]
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

// // return first true value
// func nbl(a ...bool) bool {
// 	for _, v := range a {
// 		if v {
// 			return v
// 		}
// 	}
// 	return false
// }

// // return the first non zero value
// func nzl(a ...int) int {
// 	for _, v := range a {
// 		if v > 0 {
// 			return v
// 		}
// 	}
// 	return 0
// }

// // trim name if it ends in .yaml/.yml
// func trimYaml(name string) string {
// 	if strings.HasSuffix(name, ".yaml") {
// 		name = strings.TrimSuffix(name, ".yaml")
// 	} else if strings.HasSuffix(name, ".yml") {
// 		name = strings.TrimSuffix(name, ".yml")
// 	}
// 	return name
// }

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

// https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data
// data:[<media-type>][;base64],<data>
func dataURL(mime string, raw []byte) string {
	encoded := base64.StdEncoding.EncodeToString(raw)
	d := fmt.Sprintf("data:%s;base64,%s", mime, encoded)
	return d
}

// parse s and look for agent. return app config and true if found.
func parseAgentCommand(s string) (*api.AgentTool, bool) {
	argv := shlex.Argv(s)
	at, err := conf.ParseArgs(argv)
	if err != nil || at == nil {
		return nil, false
	}
	return at, true
}

func formatArgs(args map[string]any) string {
	var sb strings.Builder
	sb.WriteString("map[")
	for k, v := range args {
		if s, ok := v.(string); ok && len(s) > maxInfoTextLen {
			sb.WriteString(fmt.Sprintf("%s: %s [%v],", k, abbreviate(s, maxInfoTextLen), len(s)))
			continue
		}
		sb.WriteString(fmt.Sprintf("%s: %v,", k, v))
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

// concat joins the s by sep.
func concat(sep byte, s ...string) string {
	return strings.Join(s, string(sep))
}
