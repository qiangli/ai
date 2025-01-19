package llm

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal"
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

	cfg := &internal.ToolConfig{}

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

	cfg := &internal.ToolConfig{}

	out, err := RunTool(cfg, ctx, name, props)
	if err != nil {
		t.Errorf("runTool error: %v", err)
		return
	}
	t.Logf("runTool name: %s out: %v\n", name, out)
}

func TestSystemToolsCheck(t *testing.T) {
	all := []string{}
	for _, v := range systemTools {
		all = append(all, v.Function.Value.Name.Value)
	}
	t.Logf("SystemTools: %v", all)

	cfg := &internal.ToolConfig{}
	ctx := context.TODO()

	for _, v := range all {
		name := v
		props := map[string]interface{}{}
		props["command"] = "which"
		props["commands"] = []string{"ls", "pwd"}
		props["args"] = []string{"ls", "pwd"}
		props["dir"] = "/tmp/"
		props["argument"] = "version"

		t.Logf("runTool name: %s, props: %v\n", name, props)

		out, err := RunTool(cfg, ctx, name, props)
		if err != nil {
			t.Errorf("runTool error: %v", err)
			return
		}
		t.Logf("runTool name: %s, out: %v\n", name, out)
	}
}

func TestExecAllowed(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	name := "exec"
	all := toolsList

	t.Logf("TestExecAllowed: %v", all)

	cfg := &internal.ToolConfig{
		// Model: &Model{
		// 	Name:   "gpt-4o-mini",
		// 	ApiKey: "sk-1234",
		// 	BaseUrl: "http://localhost:4000",
		// 	Tools: GetRestrictedTools(),
		// },
	}
	ctx := context.TODO()

	for _, v := range all {
		props := map[string]interface{}{}
		props["command"] = v
		props["args"] = []string{"--version"}

		t.Logf("runTool name: %s, props: %v\n", name, props)

		out, err := RunTool(cfg, ctx, name, props)
		if err != nil {
			t.Errorf("runTool error: %v", err)
			return
		}
		t.Logf("runTool name: %s, out: %v\n", name, out)
	}
}
