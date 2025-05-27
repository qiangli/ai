package mcp

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal/log"
)

var clientCmd = &cobra.Command{
	Use:                   "client",
	Short:                 "Client command for MCP",
	Long:                  `This command handles client operations for MCP.`,
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
}

func init() {
	var defaultPort = 5048
	if v := os.Getenv("AI_MCP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &defaultPort)
	}
	var defaultHost = "localhost"
	if v := os.Getenv("AI_MCP_HOST"); v != "" {
		fmt.Sscanf(v, "%d", &defaultPort)
	}

	// https?
	var mcpUrl = fmt.Sprintf("http://%s:%v/mcp", defaultHost, defaultPort)

	flags := clientCmd.PersistentFlags()

	// flags
	flags.VarP(newTransportValue("http", &config.Transport), "transport", "t", "Transport protocol to use: http or stdio")
	flags.StringVar(&config.McpServerUrl, "url", mcpUrl, "URL for HTTP transport")

	//
	McpCmd.AddCommand(clientCmd)
}

// Create client based on transport type
func NewMcpClient(ctx context.Context, args []string) (*client.Client, error) {

	var c *client.Client

	if config.Transport == "stdio" {
		log.Debugln("Initializing stdio client...")
		command := args[0]
		cmdArgs := args[1:]
		stdioTransport := transport.NewStdio(command, nil, cmdArgs...)

		c = client.NewClient(stdioTransport)

		if stderr, ok := client.GetStderr(c); ok {
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
	} else {
		log.Debugln("Initializing HTTP client...")
		httpTransport, err := transport.NewStreamableHTTP(config.McpServerUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP transport: %v", err)
		}
		c = client.NewClient(httpTransport)
	}

	if err := c.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start client: %v", err)
	}
	return c, nil
}

func InitMcpRequest(ctx context.Context, c *client.Client) (*mcp.InitializeResult, error) {
	// Initialize the client
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
