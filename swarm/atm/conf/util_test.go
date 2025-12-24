package conf

import (
	"reflect"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

// func TestSplitOwnerAgent(t *testing.T) {
// 	tests := []struct {
// 		name     string
// 		expected []string
// 	}{
// 		{"@agent:pack", []string{"pack", "pack"}},
// 		{"agent:pack", []string{"pack", "pack"}},
// 		{"@agent", []string{"agent", ""}},
// 		{"agent", []string{"agent", ""}},
// 		{"@", []string{"", ""}},
// 	}

// 	for i, tc := range tests {
// 		agent, sub := api.Packname(tc.name).Decode()
// 		if agent != tc.expected[0] || sub != tc.expected[1] {
// 			t.Fatalf("[%v] got: %s %s expected: %v", i, agent, sub, tc.expected)
// 		}
// 	}
// }

func TestSplit2(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{"", []string{"", ""}},
		{":", []string{"", ""}},
		{"*:", []string{"*", ""}},
		{":*", []string{"", "*"}},
		{"*:*", []string{"*", "*"}},
		{"a:b", []string{"a", "b"}},
		{"a:", []string{"a", ""}},
		{":b", []string{"", "b"}},
	}

	for i, tc := range tests {
		owner, agent := split2(tc.name, ":", "")
		if owner != tc.expected[0] || agent != tc.expected[1] {
			t.Fatalf("[%v] got: %s %s expected: %v", i, owner, agent, tc.expected)
		}
	}
}

func TestFilterTool(t *testing.T) {
	tools := []*api.ToolFunc{
		{
			Kit:  "x",
			Name: "a",
		},
		{
			Kit:  "x",
			Name: "b",
		},
		{
			Kit:  "y",
			Name: "a",
		},
	}
	xtools := []*api.ToolFunc{
		{
			Kit:  "x",
			Name: "a",
		},
		{
			Kit:  "x",
			Name: "b",
		},
	}
	atools := []*api.ToolFunc{
		{
			Kit:  "x",
			Name: "a",
		},
		{
			Kit:  "y",
			Name: "a",
		},
	}
	xbtools := []*api.ToolFunc{
		{
			Kit:  "x",
			Name: "b",
		},
	}
	tests := []struct {
		kit      string
		name     string
		expected []*api.ToolFunc
	}{
		// kit:name
		{"", "", tools},
		{"*", "", tools},
		{"", "*", tools},
		{"*", "*", tools},
		{"x", "*", xtools},
		{"*", "a", atools},
		{"x", "b", xbtools},
		{"y", "b", nil},
	}

	for i, tc := range tests {
		filtered := filterTool(tools, tc.kit, tc.name)
		if !reflect.DeepEqual(filtered, tc.expected) {
			t.Fatalf("[%v] %s:%s - expected: %v, got: %v", i, tc.kit, tc.name, tc.expected, filtered)
		}
	}
}
