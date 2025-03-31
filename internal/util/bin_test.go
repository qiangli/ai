package util

import (
	"testing"
)

func TestListCommand(t *testing.T) {
	list := ListCommands()
	t.Log(list)
	for k, v := range list {
		t.Logf("command: %s, path: %s", k, v)
	}
}
