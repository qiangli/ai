package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
)

type ServerConfig struct {
	Port         int
	Host         string
	McpServerUrl string
	Transport    string
	Debug        bool
}

var config = &ServerConfig{}

var serveCmd = &cobra.Command{
	Use:                   "serve",
	Short:                 "Start the MCP server",
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunServe(args)
	},
}

// https://github.com/mark3labs/mcp-go/blob/main/examples/everything/main.go
func RunServe(args []string) error {
	setLogLevel()

	fileLog, err := setLogOutput()
	if err != nil {
		return err
	}
	defer func() {
		if fileLog != nil {
			fileLog.Close()
		}
	}()

	cfg, err := internal.ParseConfig(args)
	if err != nil {
		return err
	}

	log.Debugf("config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.DBCred)

	vars, err := swarm.InitVars(cfg)
	if err != nil {
		return fmt.Errorf("failed to list tools: %v", err)
	}

	mcpServer := server.NewMCPServer(
		"Stargate",
		"1.0.0",
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(false),
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	toolsMap := vars.ListTools()
	for i, v := range toolsMap {
		log.Debugf("tool [%v]: %s %v\n", i, v.ID(), v)

		if err := addTool(mcpServer, vars, v); err != nil {
			log.Infof("failed to add tool [%v]: %v\n", i, err)
		}
	}

	if config.Transport == "sse" {
		baseURL := fmt.Sprintf("http://%s:%v", config.Host, config.Port)
		addr := fmt.Sprintf(":%v", config.Port)

		sse := server.NewSSEServer(mcpServer, server.WithBaseURL(baseURL))

		log.Infof("SSE server listening on :%d\n", config.Port)

		if err := sse.Start(addr); err != nil {
			return fmt.Errorf("sse server error: %v", err)
		}
	} else {
		if err := server.ServeStdio(mcpServer); err != nil {
			return fmt.Errorf("stdio server error: %v", err)
		}
	}

	return nil
}

func addMcpFlags(cmd *cobra.Command) {
	var defaultPort = 5048
	if v := os.Getenv("AI_MCP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &defaultPort)
	}

	flags := cmd.Flags()

	// flags
	flags.IntVar(&config.Port, "port", defaultPort, "Port to run the server")
	flags.StringVar(&config.Host, "host", "localhost", "Host to bind the server")

	flags.Var(newTransportValue("sse", &config.Transport), "transport", "Transport protocol to use: sse or stdio")

	flags.StringVar(&config.McpServerUrl, "mcp-server-url", "http://localhost:58080/sse", "MCP server URL")

	//
	flags.String("log", "", "Log all debugging information to a file")
	flags.Bool("verbose", false, "Show debugging information")
}

func addTool(server *server.MCPServer, vars *api.Vars, toolFunc *api.ToolFunc) error {
	toSchema := func(m map[string]any) mcp.ToolInputSchema {
		var s mcp.ToolInputSchema
		if m == nil {
			return s
		}
		if v, ok := m["type"]; ok {
			s.Type = v.(string)
		}
		if v, ok := m["properties"]; ok {
			s.Properties = v.(map[string]any)
		}
		if v, ok := m["required"]; ok {
			var required []string
			if _, ok := v.([]any); ok {
				for _, val := range v.([]any) {
					if _, ok := val.(string); !ok {
						required = append(required, val.(string))
					}
				}
			}
			s.Required = required
		}
		return s
	}

	tool := mcp.Tool{
		Name:        toolFunc.ID(),
		Description: toolFunc.Description,
		InputSchema: toSchema(toolFunc.Parameters),
	}

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		log.Debugf("Calling tool [%s] with params: %+v\n", req.Params.Name, req.Params.Arguments)

		v, err := swarm.CallTool(ctx, vars, req.Params.Name, req.GetArguments())

		log.Debugf("Tool [%s] returned: %+v\n", req.Params.Name, v)

		if err != nil {
			return nil, err
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: v.Value,
				},
			},
		}
		return result, nil
	}

	server.AddTool(tool, handler)

	return nil
}

func init() {
	addMcpFlags(serveCmd)

	// Bind the flags to viper using underscores
	serveCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	//
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	McpCmd.AddCommand(serveCmd)
}
