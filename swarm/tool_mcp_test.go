package swarm

import (
	"context"
	"testing"

	"github.com/qiangli/ai/swarm/api"
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
			t.Logf("[%s] service/name: %s %s", k, v.ID(), v.Description)
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
		t.Logf("[%v] service/name: %s %s", k, v.ID(), v.Description)
	}
}

func TestMcpCallTool(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var serverUrl = "http://localhost:58080/sse"

	// server := NewMcpServerTool(serverUrl)
	vars := &api.Vars{
		ToolRegistry: map[string]*api.ToolFunc{
			"time__convert_time": {
				Kit:  "time",
				Name: "convert_time",
			},
		},
		McpServerUrl: serverUrl,
		// McpServerTool: server,
	}

	resp, err := callMcpTool(context.TODO(), vars, "time__convert_time", map[string]interface{}{
		"source_timezone": "America/Los_Angeles",
		"time":            "16:30",
		"target_timezone": "Asia/Shanghai",
	})
	if err != nil {
		t.Errorf("call tool: %v", err)
	}
	t.Logf("response: %v", resp)
}
