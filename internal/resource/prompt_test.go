package resource

import (
	_ "embed"
	"testing"

	"github.com/qiangli/ai/internal/util"
)

func TestGetShellSystemRoleContent(t *testing.T) {
	info, err := util.CollectSystemInfo()
	if err != nil {
		t.Errorf("failed, expected nil, got %v", err)
	}
	msg, err := GetShellSystemRoleContent(info)
	if err != nil {
		t.Errorf("failed, expected nil, got %v", err)
	}
	t.Logf("\nSystem role content:\n%s\n", msg)
}

func TestGetUserHint(t *testing.T) {
	hint := GetUserHint()
	t.Logf("User hint: %s", hint)
}

func TestGetShellUserRoleContent(t *testing.T) {
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
		msg, err := GetShellUserRoleContent(
			tc.command, tc.message,
		)
		if err != nil {
			t.Errorf("failed, expected nil, got %v", err)
		}
		t.Logf("\nUser role message:\n%s\n", msg)
	}
}
