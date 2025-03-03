package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/tailscale/hujson"

	"github.com/qiangli/ai/internal/log"
)

//go:embed resource/mcp_config.jsonc
var mcpConfigData []byte

var mcpConfig = NewMcpConfig()

var mcpServerTool *McpServerTool

func init() {
	mcpConfig.Load(mcpConfigData)
	mcpConfig.ProxyUrl = os.Getenv("AI_MCP_PROXY_URL")
	mcpServerTool = NewMcpServerTool(mcpConfig)
}

func ListTools() (map[string][]*ToolFunc, error) {
	mcpConfig.ProxyUrl = os.Getenv("AI_MCP_PROXY_URL")
	return mcpServerTool.ListTools()
}

type McpConfig struct {
	ProxyUrl string `json:"baseUrl"`

	McpServers map[string]*McpServerConfig `json:"mcpServers"`
}

type McpServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

func NewMcpConfig() *McpConfig {
	return &McpConfig{
		McpServers: make(map[string]*McpServerConfig),
	}
}

func (c *McpConfig) LoadFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return c.Load(data)
}

func (c *McpConfig) Load(data []byte) error {
	hu, err := hujson.Standardize(data)
	if err != nil {
		return err
	}
	ex := expandWithDefault(string(hu))
	err = json.Unmarshal([]byte(ex), &c)
	if err != nil {
		return fmt.Errorf("unmarshal mcp config: %v", err)
	}
	return nil
}

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

func (r *McpClientSession) ListTools(ctx context.Context) ([]*ToolFunc, error) {
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := r.client.ListTools(ctx, toolsRequest)
	if err != nil {
		return nil, err
	}

	funcs := make([]*ToolFunc, 0)
	for _, v := range tools.Tools {
		funcs = append(funcs, &ToolFunc{
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

func (r *McpClientSession) GetTools(ctx context.Context, server string) ([]*FunctionConfig, error) {
	toolsRequest := mcp.ListToolsRequest{}
	tools, err := r.client.ListTools(ctx, toolsRequest)
	if err != nil {
		return nil, err
	}

	funcs := make([]*FunctionConfig, 0)
	for _, v := range tools.Tools {
		funcs = append(funcs, &FunctionConfig{
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
	for _, content := range resp.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text, nil
		} else {
			jsonBytes, _ := json.MarshalIndent(content, "", "  ")
			return string(jsonBytes), nil
		}
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

func (r *McpClientHelper) ListTools(ctx context.Context) ([]*ToolFunc, error) {
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

func NewMcpServerTool(cfg *McpConfig) *McpServerTool {
	return &McpServerTool{
		Config: cfg,
	}
}

func (r *McpServerTool) ListTools() (map[string][]*ToolFunc, error) {
	var tools = map[string][]*ToolFunc{}
	ctx := context.Background()

	if r.Config.ProxyUrl != "" {
		client := &McpClientHelper{
			ProxyUrl: r.Config.ProxyUrl,
		}
		funcs, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		// TODO better handle proxy
		for _, v := range funcs {
			parts := strings.SplitN(v.Name, "__", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid tool name: %s", v.Name)
			}
			name := parts[0]
			toolName := parts[1]
			funcs, ok := tools[name]
			v.Name = toolName
			if ok {
				tools[name] = append(funcs, v)
			} else {
				tools[name] = []*ToolFunc{v}
			}
		}
		return tools, nil
	}

	for v, cfg := range r.Config.McpServers {
		client := &McpClientHelper{
			// ProxyUrl:     r.Config.ProxyUrl,
			ServerConfig: cfg,
		}
		funcs, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		tools[v] = funcs
	}
	return tools, nil
}

func (r *McpServerTool) GetTools(server string) ([]*ToolFunc, error) {
	ctx := context.Background()
	for v, cfg := range r.Config.McpServers {
		if v == server {
			client := &McpClientHelper{
				ProxyUrl:     r.Config.ProxyUrl,
				ServerConfig: cfg,
			}
			return client.ListTools(ctx)
		}
	}
	return nil, fmt.Errorf("no such server: %s", server)
}

func (r *McpServerTool) CallTool(server, tool string, args map[string]any) (string, error) {
	ctx := context.Background()

	if r.Config.ProxyUrl != "" && !strings.HasPrefix(tool, server+"__") {
		tool = fmt.Sprintf("%s__%s", server, tool)
		client := &McpClientHelper{
			ProxyUrl: r.Config.ProxyUrl,
		}
		return client.CallTool(ctx, tool, args)
	}

	//
	for v, cfg := range r.Config.McpServers {
		if v == server {
			client := &McpClientHelper{
				ProxyUrl:     r.Config.ProxyUrl,
				ServerConfig: cfg,
			}
			resp, err := client.CallTool(ctx, tool, args)
			if err != nil {
				return "", err
			}
			if resp != "" {
				return resp, nil
			}
		}
	}
	return "", fmt.Errorf("no such server: %s", server)
}
