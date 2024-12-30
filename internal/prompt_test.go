package internal

import (
	_ "embed"
	"testing"
)

func TestGetSystemRoleContent(t *testing.T) {
	msg, err := GetSystemRoleContent()
	if err != nil {
		t.Errorf("failed, expected nil, got %v", err)
	}
	t.Logf("\nSystem role content:\n%s\n", msg)
}

func TestGetUserHint(t *testing.T) {
	hint := GetUserHint()
	t.Logf("User hint: %s", hint)
}

func TestGetUserRoleContent(t *testing.T) {
	tests := []struct {
		command string
		message string
	}{
		{"ls", "I want to list files"},
		{"man", "I want to read manual"},
		{"jq", "I want to parse JSON"},
		{"go", "I want to run Go code"},
		{"python", "I want to run Python code"},
		{"node", "I want to run Node.js code"},
		{"docker run", "I want to run Docker"},
		//
		{"", "what is unix?"},
	}
	for _, tc := range tests {
		msg, err := GetUserRoleContent(
			tc.command, tc.message,
		)
		if err != nil {
			t.Errorf("failed, expected nil, got %v", err)
		}
		t.Logf("\nUser role message:\n%s\n", msg)
	}
}
