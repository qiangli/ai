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
	t.Logf("runTool name: %s out: %v\n", name, out)
}

func TestRunToolGit(t *testing.T) {
	ctx := context.Background()

	name := "git"
	props := map[string]interface{}{
		"command": "git",
		"args":    []string{"rev-parse", "--is-inside-work-tree"},
	}
	t.Logf("runTool name: %s, props: %v\n", name, props)

	cfg := &Config{}

	out, err := RunTool(cfg, ctx, name, props)
	if err != nil {
		t.Errorf("runTool error: %v", err)
		return
	}
	t.Logf("runTool name: %s out: %v\n", name, out)
}

func TestSystemToolsCheck(t *testing.T) {
	// ctx := context.Background()

	all := []string{}
	for _, v := range SystemTools {
		all = append(all, v.Function.Value.Name.Value)
	}
	t.Logf("SystemTools: %v", all)

	for _, v := range SystemTools {
		name := v.Function.Value.Name.Value
		props := map[string]interface{}{}
		props["command"] = "which"
		props["commands"] = []string{"ls", "pwd"}
		props["args"] = []string{"ls", "pwd"}
		props["dir"] = "/tmp/"
		props["argument"] = "version"

		t.Logf("runTool name: %s, props: %v\n", name, props)

		out, err := RunTool(nil, nil, name, props)
		if err != nil {
			t.Errorf("runTool error: %v", err)
			return
		}
		t.Logf("runTool out: %v\n", out)
	}
}
