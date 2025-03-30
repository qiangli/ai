package swarm

import (
	"testing"

	"github.com/qiangli/ai/api"
)

func TestLoadToolConfig(t *testing.T) {
	base := "resource/tools"

	app := &api.AppConfig{}
	config, err := LoadToolConfig(app, base)
	if err != nil {
		t.Fatalf("failed to load tool files: %v", err)
	}

	for _, tool := range config.Tools {
		if tool.Name == "" {
			t.Fatal("tool name is empty")
		}
		if tool.Description == "" {
			t.Fatal("tool description is empty")
		}
		t.Logf("Kit: %s tool: %s - %s", tool.Kit, tool.Name, tool.Description)
	}
}

func TestLoadDefaultToolConfig(t *testing.T) {
	app := &api.AppConfig{}
	config, err := LoadDefaultToolConfig(app)
	if err != nil {
		t.Fatalf("failed to load default tool config: %v", err)
	}

	for _, tool := range config.Tools {
		if tool.Name == "" {
			t.Fatal("tool name is empty")
		}
		if tool.Description == "" {
			t.Fatal("tool description is empty")
		}
		t.Logf("Kit: %s tool: %s - %s", tool.Kit, tool.Name, tool.Description)
	}
}
