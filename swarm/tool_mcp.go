package swarm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

type McpClient struct {
	ServerConfig *api.McpServerConfig
}

func NewMcpClient(ctx context.Context, config *api.McpServerConfig) (*client.Client, error) {
	var c *client.Client

	if config.ServerUrl != "" {
		// "http"
		log.Debugln("Initializing HTTP client...")
		httpTransport, err := transport.NewStreamableHTTP(config.ServerUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP transport: %v", err)
		}
		c = client.NewClient(httpTransport)
	} else {
		// "stdio"
		log.Debugln("Initializing stdio client %+v", config.Args)
		if len(config.Args) == 0 {
			return nil, fmt.Errorf("missing args for stdio client")
		}

		stdioTransport := transport.NewStdio(config.Command, nil, config.Args...)

		c = client.NewClient(stdioTransport)

		if stderr, ok := client.GetStderr(c); ok && stderr != nil {
			go func() {
				buf := make([]byte, 4096)
				for {
					n, err := stderr.Read(buf)
					if err != nil {
						if err != io.EOF {
							log.Errorf("Error reading stderr: %v", err)
						}
						return
					}
					if n > 0 {
						log.Errorf("[Server] %s", buf[:n])
					}
				}
			}()
		}
	}

	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start client: %v", err)
	}
	return c, nil
}

func InitMcpRequest(ctx context.Context, c *client.Client) (*mcp.InitializeResult, error) {
	log.Debugln("Initializing client...")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "ai mcp client",
		Version: "0.0.1",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := c.Initialize(ctx, initRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %v", err)
	}

	log.Debugf("Connected to server: %s (version %s)\n",
		serverInfo.ServerInfo.Name,
		serverInfo.ServerInfo.Version)
	log.Debugf("Server capabilities: %+v\n", serverInfo.Capabilities)

	return serverInfo, nil
}

func (r *McpClient) ListTools(ctx context.Context) (*mcp.ListToolsResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := NewMcpClient(ctx, r.ServerConfig)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	result, err := InitMcpRequest(ctx, c)
	if err != nil {
		return nil, err
	}

	if result.Capabilities.Tools != nil {
		log.Debugln("Fetching available tools...")
		toolsRequest := mcp.ListToolsRequest{}
		toolsResult, err := c.ListTools(ctx, toolsRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to list tools: %v", err)
		} else {
			return toolsResult, nil
		}
	}

	return nil, fmt.Errorf("Tools not supported")
}

func (r *McpClient) CallTool(ctx context.Context, name string, params map[string]any) (*mcp.CallToolResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := NewMcpClient(ctx, r.ServerConfig)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	if _, err := InitMcpRequest(ctx, c); err != nil {
		return nil, err
	}

	if name == "" {
		return nil, fmt.Errorf("missing tool name")
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = params

	result, err := c.CallTool(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	return result, nil
}

type McpProxy struct {
	Config map[string]*api.McpServerConfig

	cached map[string]*mcp.ListToolsResult

	sync.Mutex
}

func NewMcpProxy(cfg map[string]*api.McpServerConfig) *McpProxy {
	return &McpProxy{
		Config: cfg,
		cached: nil,
	}
}

func (r *McpProxy) ListTools() (map[string]*mcp.ListToolsResult, error) {
	r.Lock()
	defer r.Unlock()
	if len(r.cached) != 0 {
		log.Infof("Using cached tools total %v\n", len(r.cached))
		return r.cached, nil
	}

	var tools = make(map[string]*mcp.ListToolsResult)
	ctx := context.Background()
	for v, cfg := range r.Config {
		client := &McpClient{
			ServerConfig: cfg,
		}
		funcs, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		tools[v] = funcs
	}
	r.cached = tools

	return tools, nil
}

func (r *McpProxy) GetTools(server string) (*mcp.ListToolsResult, error) {
	ctx := context.Background()
	for v, cfg := range r.Config {
		if v == server {
			client := &McpClient{
				ServerConfig: cfg,
			}
			return client.ListTools(ctx)
		}
	}
	return nil, fmt.Errorf("no such server: %s", server)
}

func (r *McpProxy) CallTool(ctx context.Context, server, tool string, args map[string]any) (*mcp.CallToolResult, error) {
	for v, cfg := range r.Config {
		if v == server {
			client := &McpClient{
				ServerConfig: cfg,
			}
			return client.CallTool(ctx, tool, args)
		}
	}
	return nil, fmt.Errorf("no such server: %s", server)
}

func listMcpTools(cfg map[string]*api.McpServerConfig) ([]*api.ToolFunc, error) {
	server := NewMcpProxy(cfg)
	result, err := server.ListTools()

	if err != nil {
		return nil, err
	}

	funcs := make([]*api.ToolFunc, 0)
	for k, v := range result {
		for _, t := range v.Tools {
			funcs = append(funcs, &api.ToolFunc{
				Type:        ToolTypeMcp,
				Kit:         k,
				Name:        t.Name,
				Description: t.Description,
				Parameters: map[string]any{
					"type":       t.InputSchema.Type,
					"properties": t.InputSchema.Properties,
					"required":   t.InputSchema.Required,
				},
			})
		}
	}

	return funcs, nil
}

func callMcpTool(ctx context.Context, mcpConf map[string]*api.McpServerConfig, vars *api.Vars, name string, args map[string]any) (string, error) {
	log.Debugf("🎖️ calling MCP tool: %s with args: %+v\n", name, args)

	// v, ok := vars.ToolRegistry[name]
	// if !ok {
	// 	return "", fmt.Errorf("no such mcp tool: %s", name)
	// }

	tools, err := vars.Config.ToolLoader(name)
	if err != nil {
		return "", fmt.Errorf("no such mcp tool: %s", name)
	}
	v := tools[0]

	server := NewMcpProxy(mcpConf)

	parts := strings.SplitN(v.ID(), "__", 2)

	result, err := server.CallTool(ctx, parts[0], parts[1], args)
	if err != nil {
		return "", err
	}

	out := func(result *mcp.CallToolResult) string {
		for _, content := range result.Content {
			if textContent, ok := content.(mcp.TextContent); ok {
				return textContent.Text
			}
		}
		//TODO
		return "tool call returned no text output"
	}

	return out(result), err
}
