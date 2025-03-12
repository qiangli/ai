package swarm

import (
	"testing"
)

func TestMcpListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"

	server := NewMcpServerTool(serverUrl)

	tools, err := server.ListTools()
	if err != nil {
		t.Errorf("list tools: %v", err)
	}

	for k, tool := range tools {
		for _, v := range tool {
			t.Logf("[%s] service: %s func: %s %s", k, v.Service, v.Func, v.Description)
		}
	}
	t.Logf("Total: %v", len(tools))
}

func TestMcpGetTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"

	server := NewMcpServerTool(serverUrl)

	tools, err := server.GetTools("time")
	if err != nil {
		t.Errorf("list tools: %v", err)
	}

	for k, v := range tools {
		t.Logf("[%v] service: %s func: %s %s", k, v.Service, v.Func, v.Description)
	}
}

func TestMcpCallTool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"

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
