package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestSseListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	mcpUrl := "http://localhost:5048/sse"
	client, err := client.NewSSEMCPClient(mcpUrl)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the client
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	initResult, err := client.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Logf(
		"Initialized with server: %s %s\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// Test Ping
	if err := client.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
	t.Log("Ping successful")

	// List Tools
	fmt.Println("Listing available tools...")
	listReq := mcp.ListToolsRequest{}
	tools, err := client.ListTools(ctx, listReq)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	t.Log("done.")
}

func TestSseCallTool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	mcpUrl := "http://localhost:5048/sse"
	client, err := client.NewSSEMCPClient(mcpUrl)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start the client
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Failed to start client: %v", err)
	}

	// Initialize
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}

	initResult, err := client.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	t.Logf(
		"Initialized with server: %s %s\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// Call a tool
	callTests := []struct {
		name       string
		toolName   string
		arguments  map[string]interface{}
		wantResult bool
	}{
		{
			name:     "Convert time",
			toolName: "time__convert_time",
			arguments: map[string]interface{}{
				"source_timezone": "America/Los_Angeles",
				"time":            "16:30",
				"target_timezone": "Asia/Shanghai",
			},
			wantResult: true,
		},
		{
			name:     "Search on DuckDuckGo",
			toolName: "ddg__search",
			arguments: map[string]interface{}{
				"query":       "weather in sfo ca",
				"max_results": 1,
			},
			wantResult: true,
		},
	}

	for _, test := range callTests {
		t.Run(test.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Name = test.toolName
			req.Params.Arguments = test.arguments

			result, err := client.CallTool(ctx, req)
			if err != nil {
				t.Fatalf("Failed to call: %+v %v", req, err)
			}
			for _, content := range result.Content {
				if textContent, ok := content.(mcp.TextContent); ok {
					fmt.Println(textContent.Text)
				} else {
					jsonBytes, _ := json.MarshalIndent(content, "", "  ")
					fmt.Println(string(jsonBytes))
				}
			}
		})
	}
}

func TestStdioListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	// go run ./cmd/ai /mcp serve --transport stdio
	c, err := client.NewStdioMCPClient(
		"go",
		[]string{},
		"run",
		"./cmd/ai",
		"/mcp",
		"serve",
		"--transport",
		"stdio",
		"--verbose",
	)

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize the client
	fmt.Println("Initializing client...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "example-client",
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}
	t.Logf(
		"Initialized with server: %s %s\n\n",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// List Tools
	fmt.Println("Listing available tools...")
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}
	for _, tool := range tools.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}
	t.Log("done.")
}
