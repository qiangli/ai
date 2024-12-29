package internal

import (
	"testing"
)

func TestListCommand(t *testing.T) {
	list, err := ListCommands(true)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	t.Log(list)
	t.Logf("total: %v", len(list))
}
