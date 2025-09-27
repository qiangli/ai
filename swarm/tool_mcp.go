package swarm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

type McpClient struct {
	cfg *api.ConnectorConfig
}

func NewMcpClient(cfg *api.ConnectorConfig) *McpClient {
	return &McpClient{
		cfg: cfg,
	}
}

func (r *McpClient) Connect(ctx context.Context) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-client",
		Version: "0.0.1",
	}, nil)

	return client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: r.cfg.BaseUrl,
	}, nil)
}

func ListMcpTools(tc *api.ToolsConfig) ([]*api.ToolFunc, error) {
	ctx := context.Background()

	if tc.Connector == nil || tc.Connector.BaseUrl == "" {
		return nil, fmt.Errorf("Invalid mcp config. Missing URL")
	}

	log.GetLogger(ctx).Debugf("Connecting to MCP server at %s", tc.Connector.BaseUrl)

	client := NewMcpClient(tc.Connector)
	session, err := client.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	log.GetLogger(ctx).Debugf("Connected to server: session ID: %s)", session.ID())

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	funcs := make([]*api.ToolFunc, 0)
	for _, v := range result.Tools {
		params, err := structToMap(v.InputSchema)
		if err != nil {
			return nil, err
		}
		tool := &api.ToolFunc{
			Kit:         tc.Kit,
			Type:        api.ToolTypeMcp,
			Name:        v.Name,
			Description: v.Description,
			Parameters:  params,
			Provider:    nvl(tc.Connector.Provider, tc.Provider),
			BaseUrl:     nvl(tc.Connector.BaseUrl, tc.BaseUrl),
			ApiKey:      nvl(tc.Connector.ApiKey, tc.ApiKey),
			//
			Config: tc,
		}
		funcs = append(funcs, tool)
	}

	return funcs, nil
}

func callMcpTool(ctx context.Context, tf *api.ToolFunc, vars *api.Vars, name string, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("🎖️ calling MCP tool: %s with args: %+v\n", name, args)

	if tf.Config == nil || tf.Config.Connector == nil || tf.Config.Connector.BaseUrl == "" {
		return "", fmt.Errorf("mcp not configured: %s", name)
	}

	client := NewMcpClient(tf.Config.Connector)
	session, err := client.Connect(ctx)
	if err != nil {
		return "", err
	}
	defer session.Close()

	log.GetLogger(ctx).Debugf("Connected to mcp server session ID: %s)", session.ID())

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tf.Name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}

	for _, content := range result.Content {
		if v, ok := content.(*mcp.TextContent); ok {
			return v.Text, nil
		}
	}

	return "", fmt.Errorf("No response")
}
