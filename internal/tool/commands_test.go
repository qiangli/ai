package tool

import (
	"context"
	"testing"
)

func TestRunMan(t *testing.T) {
	out, err := runMan("zic")
	if err != nil {
		t.Errorf("runMan error: %v", err)
		return
	}
	t.Logf("runMan out:\n%v", out)
}

func TestRunTool(t *testing.T) {
	ctx := context.Background()

	name := "man"
	props := map[string]interface{}{
		"command": "zic",
	}
	t.Logf("runTool name: %s, props: %v\n", name, props)

	cfg := &Config{}

	out, err := RunTool(cfg, ctx, name, props)
	if err != nil {
		t.Errorf("runTool error: %v", err)
		return
	}
	t.Logf("runTool out:\n%v", out)
}

func TestListCommand(t *testing.T) {
	list, err := ListCommands(true)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	t.Log(list)
	t.Logf("total: %v", len(list))
}
