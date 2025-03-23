package util

import (
	"testing"
)

func TestListCommand(t *testing.T) {
	list := ListCommands()
	t.Log(list)
	for _, cmd := range list {
		t.Logf("command: %s, path: %s", cmd[0], cmd[1])
	}
}
