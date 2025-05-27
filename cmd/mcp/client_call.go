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
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		name, _ := cmd.Flags().GetString("name")
		arguments, _ := cmd.Flags().GetString("arguments")
		server, _ := cmd.Flags().GetString("server")
		cmdArgs, _ := cmd.Flags().GetString("args")
		config.Server = server
		config.Args = strings.Fields(cmdArgs)

		var params map[string]any
		if arguments != "" {
			if err := json.Unmarshal([]byte(arguments), &params); err != nil {
				return err
			}
		}
		if params == nil {
			params = make(map[string]any)
		}

		// command line args: name=value
		for _, nv := range args {
			parts := strings.SplitN(nv, "=", 2)
			var n = parts[0]
			var v string
			if len(parts) > 1 {
				v = parts[1]
			}
			params[n] = v
		}

		result, err := CallTool(ctx, config, name, params)
		if err != nil {
			return err
		}
		printToolResult(result)
		return nil
	},
}

func init() {
	flags := clientCallCmd.Flags()
	flags.String("name", "", "Tool name")
	flags.String("arguments", "", "Tool arguments in json format")

	clientCmd.AddCommand(clientCallCmd)
}

func CallTool(ctx context.Context, config *ServerConfig, name string, params map[string]any) (*mcp.CallToolResult, error) {
	c, err := NewMcpClient(ctx, config)
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
