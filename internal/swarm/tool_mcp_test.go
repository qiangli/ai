package swarm

import (
	"os"
	"testing"
)

func TestMcpConfigLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	os.Setenv("AI_SQL_DB_NAME", "doc")
	os.Setenv("AI_SQL_DB_USERNAME", "admin")
	os.Setenv("AI_SQL_DB_PASSWORD", "password")

	var cfg = NewMcpConfig("")
	err := cfg.LoadFile("resource/mcp_config.jsonc")
	if err != nil {
		t.Errorf("load mcp config: %v", err)
	}
}

func TestMcpListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"
	// var cfg = NewMcpConfig(serverUrl)
	// err := cfg.LoadFile("resource/mcp_config.jsonc")
	// if err != nil {
	// 	t.Errorf("load mcp config: %v", err)
	// }

	server := NewMcpServerTool(serverUrl)

	tools, err := server.ListTools()
	if err != nil {
		t.Errorf("list tools: %v", err)
	}

	for k, tool := range tools {
		for _, v := range tool {
			t.Logf("server: %s tools: %s %s\n", k, v.Name, v.Description)
		}
	}
	t.Logf("Total: %v", len(tools))
}

func TestMcpGetTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"
	// var cfg = NewMcpConfig(serverUrl)
	// err := cfg.LoadFile("resource/mcp_config.jsonc")
	// if err != nil {
	// 	t.Errorf("load mcp config: %v", err)
	// }

	server := NewMcpServerTool(serverUrl)

	tools, err := server.GetTools("time")
	if err != nil {
		t.Errorf("list tools: %v", err)
	}

	for _, v := range tools {
		t.Logf("tools: %s %s\n", v.Name, v.Description)
	}
}

func TestMcpCallTool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"
	// var cfg = NewMcpConfig(serverUrl)
	// err := cfg.LoadFile("resource/mcp_config.jsonc")
	// if err != nil {
	// 	t.Errorf("load mcp config: %v", err)
	// }

	server := NewMcpServerTool(serverUrl)

	resp, err := server.CallTool("time", "convert_time", map[string]interface{}{
		"source_timezone": "America/Los_Angeles",
		"time":            "16:30",
		"target_timezone": "Asia/Shanghai",
	})
	if err != nil {
		t.Errorf("call tool: %v", err)
	}
	t.Logf("response: %v", resp)
}
