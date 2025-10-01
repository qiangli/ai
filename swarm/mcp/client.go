package mcp

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qiangli/ai/swarm/api"
)

type bearerTokenTransport struct {
	token     string
	transport http.RoundTripper
}

func (b *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.transport.RoundTrip(req)
}

type McpClient struct {
	cfg *api.ConnectorConfig
}

func NewMcpClient(cfg *api.ConnectorConfig) *McpClient {
	return &McpClient{
		cfg: cfg,
	}
}

func (r *McpClient) Connect(ctx context.Context, token string) (*mcp.ClientSession, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "mcp-client",
		Version: "0.0.1",
	}, nil)

	transport := &http.Transport{}
	httpClient := &http.Client{
		Transport: &bearerTokenTransport{
			token:     token,
			transport: transport,
		},
	}

	return client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   r.cfg.BaseUrl,
		HTTPClient: httpClient,
	}, nil)
}
