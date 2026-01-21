package api

import (
	"fmt"
	"testing"
)

func TestNewKitname(t *testing.T) {
	tests := []struct {
		kit      string
		name     string
		expected string
	}{
		{"agent", "ed", "agent__ed__ed"},
	}
	for _, tt := range tests {
		tid := NewKitname(tt.kit, tt.name).ID()
		if tid != tt.expected {
			t.Fatalf("expeted: %s got %s", tt.expected, tid)
		}
	}
}

func TestKitname(t *testing.T) {
	tests := []struct {
		input string
		// expected
		kit  string
		name string
	}{
		{
			input: "@example",
			kit:   "agent",
			name:  "example/example",
		},
		// root
		{
			input: "@",
			kit:   "agent",
			name:  "/",
		},
		{
			input: "/agent:example",
			kit:   "agent",
			name:  "example/example",
		},
		{
			input: "example,",
			kit:   "agent",
			name:  "example/example",
		},
		{
			input: "/",
			kit:   "",
			name:  "/",
		},
		{
			input: "/bin/ls",
			kit:   "",
			name:  "/bin/ls",
		},
		{
			input: "/bin",
			kit:   "",
			name:  "/bin",
		},
		{
			input: "ls -al",
			kit:   "",
			name:  "ls -al",
		},
		// kit__name
		{
			input: "kit__name",
			kit:   "kit",
			name:  "name",
		},
		// kit:*
		{
			input: "/kit:*",
			kit:   "kit",
			name:  "*",
		},
		// kit:name
		{
			input: "/kit:name",
			kit:   "kit",
			name:  "name",
		},
		// agent:name
		{
			input: "/agent:name",
			kit:   "agent",
			name:  "name/name",
		},
		// @name
		{
			input: "/@name",
			kit:   "agent",
			name:  "name/name",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test Case %d", i), func(t *testing.T) {
			kit, name := Kitname(tt.input).Decode()
			if tt.kit != kit || tt.name != name {
				t.Errorf("ActionKitname = %q %q, want %q %q", kit, name, tt.kit, tt.name)
			}
		})
	}
}
