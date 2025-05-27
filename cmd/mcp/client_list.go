package mcp

import (
	"context"
	"encoding/json"
	"fmt"
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

		return ListTools(args)
	},
}

func init() {
	//
	clientCmd.AddCommand(clientListCmd)
}

func ListTools(args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := NewMcpClient(ctx, args)
	if err != nil {
		return err
	}
	defer c.Close()

	c.OnNotification(func(notification mcp.JSONRPCNotification) {
		log.Debugf("Received notification: %s\n", notification.Method)
	})

	result, err := InitMcpRequest(ctx, c)
	if err != nil {
		return err
	}

	if result.Capabilities.Tools != nil {
		log.Debugln("Fetching available tools...")
		toolsRequest := mcp.ListToolsRequest{}
		toolsResult, err := c.ListTools(ctx, toolsRequest)
		if err != nil {
			return fmt.Errorf("failed to list tools: %v", err)
		} else {
			log.Infof("Tools available: %d\n", len(toolsResult.Tools))
			for i, tool := range toolsResult.Tools {
				schema, _ := json.MarshalIndent(tool.InputSchema, "", "  ")
				log.Infof("  %d. %s - %s\n%s\n\n", i+1, tool.Name, tool.Description, string(schema))
			}
		}
	}

	// List available resources if the server supports them
	if result.Capabilities.Resources != nil {
		log.Debugln("Fetching available resources...")
		resourcesRequest := mcp.ListResourcesRequest{}
		resourcesResult, err := c.ListResources(ctx, resourcesRequest)
		if err != nil {
			return fmt.Errorf("failed to list resources: %v", err)
		} else {
			log.Debugf("Resources available: %d\n", len(resourcesResult.Resources))
			for i, resource := range resourcesResult.Resources {
				log.Infof("  %d. %s - %s\n", i+1, resource.URI, resource.Name)
			}
		}
	}

	log.Debugln("Done. shutting down...")
	return nil
}
