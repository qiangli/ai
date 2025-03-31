package swarm

import (
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestLoadToolsConfig(t *testing.T) {
	base := "resource/tools"

	app := &api.AppConfig{}
	config, err := LoadToolsConfig(app, base)
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

func TestLoadDefaultToolsConfig(t *testing.T) {
	app := &api.AppConfig{}
	config, err := LoadDefaultToolsConfig(app)
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
