package swarm

import (
	"testing"
)

func TestMcpConfigLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var cfg = NewMcpConfig()
	err := cfg.LoadFile("resource/mcp_config.json")
	if err != nil {
		t.Errorf("load mcp config: %v", err)
	}
}

func TestMcpListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var cfg = NewMcpConfig()
	err := cfg.LoadFile("resource/mcp_config.json")
	if err != nil {
		t.Errorf("load mcp config: %v", err)
	}

	server := NewMcpServer()

	tools, err := server.ListTools()
	if err != nil {
		t.Errorf("list tools: %v", err)
	}

	for _, tool := range tools {
		t.Logf("tool: %v", tool)
	}
}
