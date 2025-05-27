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

var clientCallCmd = &cobra.Command{
	Use:   "call [param=value...]",
	Short: "Call tool",
	Long:  `This command calls tool.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		setLogLevel()

		return CallTool(cmd, args)
	},
}

func init() {
	flags := clientCallCmd.Flags()
	flags.String("name", "", "Tool name")
	flags.String("arguments", "", "Tool arguments in json format")

	clientCmd.AddCommand(clientCallCmd)
}

func CallTool(cmd *cobra.Command, args []string) error {
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

	if _, err := InitMcpRequest(ctx, c); err != nil {
		return err
	}

	name, _ := cmd.Flags().GetString("name")
	arguments, _ := cmd.Flags().GetString("arguments")

	if name == "" {
		return fmt.Errorf("missing tool name")
	}

	var paramArgs map[string]any
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &paramArgs); err != nil {
			return err
		}
	}
	if paramArgs == nil {
		paramArgs = make(map[string]any)
	}

	// command line args: name=value
	for _, nv := range args {
		parts := strings.SplitN(nv, "=", 2)
		var n = parts[0]
		var v string
		if len(parts) > 1 {
			v = parts[1]
		}
		paramArgs[n] = v
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = paramArgs

	result, err := c.CallTool(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}
	printToolResult(result)

	return nil
}

func printToolResult(result *mcp.CallToolResult) {
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			log.Infoln(textContent.Text)
		} else {
			jsonBytes, _ := json.MarshalIndent(content, "", "  ")
			log.Infoln(string(jsonBytes))
		}
	}
}
