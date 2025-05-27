package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal/log"
)

var clientListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tools",
	Long:  `This command lists all tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		setLogLevel()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server, _ := cmd.Flags().GetString("server")
		cmdArgs, _ := cmd.Flags().GetString("args")
		config.Server = server
		config.Args = strings.Fields(cmdArgs)
		result, err := ListTools(ctx, config)
		if err != nil {
			return err
		}
		log.Infof("Tools available: %d\n", len(result.Tools))
		for i, tool := range result.Tools {
			schema, _ := json.MarshalIndent(tool.InputSchema, "", "  ")
			log.Infof("  %d. %s - %s\n%s\n\n", i+1, tool.Name, tool.Description, string(schema))
		}
		return nil
	},
}

func init() {
	//
	clientCmd.AddCommand(clientListCmd)
}

func ListTools(ctx context.Context, config *ServerConfig) (*mcp.ListToolsResult, error) {

	c, err := NewMcpClient(ctx, config)
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
