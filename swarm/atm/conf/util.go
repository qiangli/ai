package conf

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

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

// // split @agent[:owner]
// // return owner, agent
// func splitOwnerAgent(s string) (string, string) {
// 	s = strings.TrimPrefix(s, "@")
// 	agent, owner := split2(s, ":", "")
// 	return owner, agent
// }

func filterTool(tools []*api.ToolFunc, kit, name string) []*api.ToolFunc {
	// return true if a is blank, star, or a/b are equal
	eq := func(s, n string) bool {
		return s == "" || s == "*" || s == n
	}

	var filtered []*api.ToolFunc
	for _, v := range tools {
		if eq(kit, v.Kit) && eq(name, v.Name) {
			filtered = append(filtered, v)
		}
	}

	return filtered
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

// return the first non empty string
func nvl(a ...string) string {
	for _, v := range a {
		if v != "" {
			return v
		}
	}
	return ""
}

// return first true value
func nbl(a ...*bool) bool {
	for _, v := range a {
		if v != nil {
			return *v
		}
	}
	return false
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

func normalizedName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	return strings.ReplaceAll(name, " ", "-")
}

func trimExt(s string) string {
	return strings.TrimSuffix(s, path.Ext(s))
}

// convert filename to agent pack name
func Packname(s string) string {
	return normalizedName(trimExt(s))
}

// convert filename to toolkit name
func Kitname(s string) string {
	return normalizedName(trimExt(s))
}

// convert filename to model alias
func modelName(s string) string {
	return normalizedName(trimExt(s))
}
