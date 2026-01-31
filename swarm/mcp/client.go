package mcp

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ConnectorConfig struct {
	// mcp | ssh ...
	// Proto string `yaml:"proto"`

	// mcp stdin/stdout
	// https://github.com/modelcontextprotocol/servers/tree/main
	// Command string `yaml:"command"`
	// Args    string `yaml:"args"`

	// ssh://user@example.com:2222/user/home
	// git@github.com:owner/repo.git
	// postgres://dbuser:secret@db.example.com:5432/mydb?sslmode=require
	// https://drive.google.com/drive/folders
	// mailto:someone@example.com

	// optional as of now
	Provider string `yaml:"provider"`

	BaseUrl string `yaml:"base_url"`
	// name of api lookup key
	ApiKey string `yaml:"api_key"`
}

type bearerTokenTransport struct {
	token     string
	transport http.RoundTripper
}

func (b *bearerTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.transport.RoundTrip(req)
}

type McpClient struct {
	cfg *ConnectorConfig
}

func NewMcpClient(cfg *ConnectorConfig) *McpClient {
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
