package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	// "github.com/tailscale/hujson"

	"github.com/qiangli/ai/api"
	"github.com/qiangli/ai/internal/log"
)

func ListMcpTools(serverUrl string) (map[string][]*api.ToolFunc, error) {
	tools := NewMcpServerTool(serverUrl)
	return tools.ListTools()
}

type McpConfig struct {
	ServerUrl string `json:"serverUrl"`
}

type McpServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func NewMcpConfig(serverUrl string) *McpConfig {
	return &McpConfig{
		ServerUrl: serverUrl,
	}
}

// func (c *McpConfig) LoadFile(filename string) error {
// 	data, err := os.ReadFile(filename)
// 	if err != nil {
// 		return err
// 	}
// 	return c.Load(data)
// }

// func (c *McpConfig) Load(data []byte) error {
// 	hu, err := hujson.Standardize(data)
// 	if err != nil {
// 		return err
// 	}
// 	ex := expandWithDefault(string(hu))
// 	err = json.Unmarshal([]byte(ex), &c)
// 	if err != nil {
// 		return fmt.Errorf("unmarshal mcp config: %v", err)
// 	}
// 	return nil
// }

type MCPClient interface {
	ListTools(context.Context, mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	CallTool(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	Initialize(context.Context, mcp.InitializeRequest) (*mcp.InitializeResult, error)
	Close() error
}

type McpClientSession struct {
	baseUrl string
	cfg     *McpServerConfig

	client MCPClient
}

func (r *McpClientSession) createStdioClient(ctx context.Context) (MCPClient, error) {
	client, err := client.NewStdioMCPClient(
		r.cfg.Command,
		os.Environ(),
		r.cfg.Args...,
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *McpClientSession) createSSEClient(ctx context.Context, baseUrl string) (MCPClient, error) {
	client, err := client.NewSSEMCPClient(baseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSE client: %v", err)
	}
	defer client.Close()

	// Start the client
	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start SSE client: %v", err)
	}
	return client, nil
}

func (r *McpClientSession) Connect(ctx context.Context) error {
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "swarm-client",
		Version: "1.0.0",
	}

	var err error
	var client MCPClient
	var result *mcp.InitializeResult

	if r.baseUrl != "" {
		client, err = r.createSSEClient(ctx, r.baseUrl)
	} else {
		client, err = r.createStdioClient(ctx)
	}
	if err != nil {
		return err
	}

	result, err = client.Initialize(ctx, initRequest)
	if err != nil {
		return err
	}

	r.client = client

	log.Debugf("Initialized: %s %s\n", result.ServerInfo.Name, result.ServerInfo.Version)
	return nil
}

func (r *McpClientSession) ListTools(ctx context.Context) ([]*api.ToolFunc, error) {
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := r.client.ListTools(ctx, toolsRequest)
	if err != nil {
		return nil, err
	}

	funcs := make([]*api.ToolFunc, 0)
	for _, v := range tools.Tools {
		parts := strings.SplitN(v.Name, "__", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tool name: %s", v.Name)
		}
		funcs = append(funcs, &api.ToolFunc{
			Type:        ToolTypeMcp,
			Kit:         parts[0],
			Name:        parts[1],
			Description: v.Description,
			Parameters: map[string]any{
				"type":       v.InputSchema.Type,
				"properties": v.InputSchema.Properties,
				"required":   v.InputSchema.Required,
			},
		})
	}
	return funcs, nil
}

func (r *McpClientSession) GetTools(ctx context.Context, server string) ([]*api.FunctionConfig, error) {
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := r.client.ListTools(ctx, toolsRequest)
	if err != nil {
		return nil, err
	}

	funcs := make([]*api.FunctionConfig, 0)
	for _, v := range tools.Tools {
		funcs = append(funcs, &api.FunctionConfig{
			Name:        v.Name,
			Description: v.Description,
			Parameters: map[string]any{
				"type":       v.InputSchema.Type,
				"properties": v.InputSchema.Properties,
				"required":   v.InputSchema.Required,
			},
		})
	}
	return funcs, nil
}

func (r *McpClientSession) CallTool(ctx context.Context, tool string, args map[string]any) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = tool
	req.Params.Arguments = args

	resp, err := r.client.CallTool(ctx, req)
	if err != nil {
		return "", err
	}
	for _, v := range resp.Content {
		jsonBytes, _ := json.MarshalIndent(v, "", "  ")
		return string(jsonBytes), nil
	}
	return "", nil
}

func (r *McpClientSession) Close() error {
	if r.client == nil {
		return nil
	}
	return r.client.Close()
}

type McpClientHelper struct {
	ProxyUrl     string
	ServerConfig *McpServerConfig
}

func (r *McpClientHelper) ListTools(ctx context.Context) ([]*api.ToolFunc, error) {
	clientSession := &McpClientSession{
		baseUrl: r.ProxyUrl,
		cfg:     r.ServerConfig,
	}
	if err := clientSession.Connect(ctx); err != nil {
		return nil, err
	}
	defer clientSession.Close()

	return clientSession.ListTools(ctx)
}

func (r *McpClientHelper) CallTool(ctx context.Context, tool string, args map[string]any) (string, error) {
	clientSession := &McpClientSession{
		baseUrl: r.ProxyUrl,
		cfg:     r.ServerConfig,
	}
	if err := clientSession.Connect(ctx); err != nil {
		return "", err
	}
	defer clientSession.Close()

	return clientSession.CallTool(ctx, tool, args)
}

type McpServerTool struct {
	Config *McpConfig
}

func NewMcpServerTool(serverUrl string) *McpServerTool {
	return &McpServerTool{
		Config: NewMcpConfig(serverUrl),
	}
}

// ListTools retrieves the list of tools from the MCP server keyed by the server name.
func (r *McpServerTool) ListTools() (map[string][]*api.ToolFunc, error) {
	var kits = map[string][]*api.ToolFunc{}
	ctx := context.Background()

	if r.Config.ServerUrl != "" {
		client := &McpClientHelper{
			ProxyUrl: r.Config.ServerUrl,
		}
		funcs, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		for _, v := range funcs {
			kit := v.Kit
			funcs, ok := kits[kit]
			if ok {
				kits[kit] = append(funcs, v)
			} else {
				kits[kit] = []*api.ToolFunc{v}
			}
		}
		return kits, nil
	}
	return kits, nil
}

func (r *McpServerTool) GetTools(server string) ([]*api.ToolFunc, error) {
	tools, err := r.ListTools()
	if err != nil {
		return nil, err
	}
	for v, tools := range tools {
		if v == server {
			return tools, nil
		}
	}
	return nil, fmt.Errorf("no such server: %s", server)
}

func (r *McpServerTool) CallTool(ctx context.Context, tool string, args map[string]any) (string, error) {
	if r.Config.ServerUrl == "" {
		return "", fmt.Errorf("server url not configured")
	}

	client := &McpClientHelper{
		ProxyUrl: r.Config.ServerUrl,
	}

	return client.CallTool(ctx, tool, args)
}

func callMcpTool(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	log.Debugf("🎖️ calling MCP tool: %s with args: %+v\n", name, args)

	v, ok := vars.ToolRegistry[name]
	if !ok {
		return "", fmt.Errorf("no such mcp tool: %s", name)
	}

	server := NewMcpServerTool(vars.McpServerUrl)

	out, err := server.CallTool(ctx, v.ID(), args)
	return out, err
}
