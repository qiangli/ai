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
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

type ServerConfig struct {
	Port         int
	Host         string
	McpServerUrl string
	Debug        bool
}

var port int
var host string
var mcpServerUrl string

var mcpServer *server.MCPServer

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunServe(args)
	},
}

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

	log.Debugf("config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.Db)

	//
	toolsMap, err := agent.ListServiceTools(mcpServerUrl)
	if err != nil {
		return fmt.Errorf("failed to list tools: %v", err)
	}

	mcpServer = server.NewMCPServer(
		"Stargate",
		"1.0.0",
		// server.WithResourceCapabilities(true, true),
		// server.WithPromptCapabilities(true),
		server.WithLogging(),
	)

	// Add tools
	var app = &internal.AppConfig{}
	vars, err := swarm.InitVars(app)
	if err != nil {
		return fmt.Errorf("failed to initialize vars: %v", err)
	}
	vars.ToolRegistry = make(map[string]*api.ToolFunc)
	vars.McpServerUrl = mcpServerUrl

	for i, v := range toolsMap {
		log.Debugf("tool [%v]: %s %v\n", i, v.ID(), v)

		if err := addTool(vars, v); err != nil {
			log.Infof("failed to add tool [%v]: %v", i, err)
		}

		if _, ok := vars.ToolRegistry[v.ID()]; ok {
			log.Infof("tool [%s] already exists, skipping...\n", v.ID())
			continue
		}
		vars.ToolRegistry[v.ID()] = v
	}

	baseURL := fmt.Sprintf("http://%s:%v", host, port)
	addr := fmt.Sprintf(":%v", port)

	sse := server.NewSSEServer(mcpServer, server.WithBaseURL(baseURL))

	log.Infof("SSE server listening on :%d\n", port)

	if err := sse.Start(addr); err != nil {
		return fmt.Errorf("server error: %v", err)
	}

	return nil
}

func addFlags(cmd *cobra.Command) {
	var defaultPort = 58888
	if v := os.Getenv("AI_MCP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &defaultPort)
	}

	flags := cmd.Flags()

	// flags
	flags.IntVar(&port, "port", defaultPort, "Port to run the server")
	flags.StringVar(&host, "host", "localhost", "Host to bind the server")
	flags.StringVar(&mcpServerUrl, "mcp-server-url", "http://localhost:58080/sse", "MCP server URL")

	//
	flags.String("log", "", "Log all debugging information to a file")
	flags.Bool("verbose", false, "Show debugging information")
	flags.Bool("trace", false, "Trace API calls")
}

func addTool(vars *swarm.Vars, toolFunc *api.ToolFunc) error {
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

		v, err := vars.CallTool(ctx, req.Params.Name, req.Params.Arguments)

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

	mcpServer.AddTool(tool, handler)

	return nil
}

func init() {
	addFlags(serveCmd)

	// Bind the flags to viper using underscores
	serveCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	// Bind the flags to viper using dots
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	McpCmd.AddCommand(serveCmd)
}
