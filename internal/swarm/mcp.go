package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
	"github.com/tailscale/hujson"
)

//go:embed resource/mcp_config.jsonc
var mcpConfigData []byte

var mcpConfig = NewMcpConfig()

func init() {
	mcpConfig.Load(mcpConfigData)
}

type McpConfig struct {
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
	err = json.Unmarshal(hu, &c)
	if err != nil {
		return fmt.Errorf("unmarshal mcp config: %v", err)
	}
	return nil
}

type ToolResponse = mcp.ToolResponse

type McpClientSession struct {
	cfg *McpServerConfig

	cmd    *exec.Cmd
	client *mcp.Client
}

func (r *McpClientSession) Connect(ctx context.Context) error {
	cmd := exec.Command(r.cfg.Command, r.cfg.Args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	clientTransport := stdio.NewStdioServerTransportWithIO(stdout, stdin)
	client := mcp.NewClient(clientTransport)

	if _, err := client.Initialize(ctx); err != nil {
		return err
	}

	r.cmd = cmd
	r.client = client

	return nil
}

func (r *McpClientSession) ListTools(ctx context.Context) ([]*FunctionConfig, error) {
	tools, err := r.client.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	funcs := make([]*FunctionConfig, len(tools.Tools))
	for _, tool := range tools.Tools {
		desc := ""
		if tool.Description != nil {
			desc = *tool.Description
		}

		params, err := structToMap(tool.InputSchema)
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, &FunctionConfig{
			Name:        tool.Name,
			Description: desc,
			Parameters:  params,
		})
	}
	return funcs, nil
}

func (r *McpClientSession) CallTool(ctx context.Context, tool string, args any) (string, error) {
	resp, err := r.client.CallTool(ctx, tool, args)
	if err != nil {
		return "", err
	}
	for _, content := range resp.Content {
		if content.TextContent != nil {
			return content.TextContent.Text, nil
		}
	}
	return "", nil
}

func (r *McpClientSession) Close() error {
	if r.cmd == nil {
		return nil
	}
	return r.cmd.Process.Kill()
}

type McpClient struct {
	ServerConfig *McpServerConfig
}

func (r *McpClient) ListTools(ctx context.Context) ([]*FunctionConfig, error) {
	clientSession := &McpClientSession{
		cfg: r.ServerConfig,
	}
	if err := clientSession.Connect(ctx); err != nil {
		return nil, err
	}
	defer clientSession.Close()

	return clientSession.ListTools(ctx)
}

func (r *McpClient) CallTool(ctx context.Context, tool string, args any) (string, error) {
	clientSession := &McpClientSession{
		cfg: r.ServerConfig,
	}
	if err := clientSession.Connect(ctx); err != nil {
		return "", err
	}
	defer clientSession.Close()

	return clientSession.CallTool(ctx, tool, args)
}

type McpServer struct {
	Config *McpConfig
}

func NewMcpServer() *McpServer {
	return &McpServer{
		Config: mcpConfig,
	}
}

func (r *McpServer) ListTools() ([]*FunctionConfig, error) {
	var tools []*FunctionConfig
	ctx := context.Background()
	for _, cfg := range r.Config.McpServers {
		client := &McpClient{
			ServerConfig: cfg,
		}
		funcs, err := client.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		tools = append(tools, funcs...)
	}
	return tools, nil
}

func (r *McpServer) CallTool(tool string, args any) (string, error) {
	ctx := context.Background()
	for _, cfg := range r.Config.McpServers {
		if tool == cfg.Command {
			client := &McpClient{
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
	return "", fmt.Errorf("no such tool: %s", tool)
}
