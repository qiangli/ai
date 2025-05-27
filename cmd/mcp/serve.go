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

// var updated during build
var ServerName = "Stargate"
var ServerVersion = "0.0.1"

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

		return Serve(args)
	},
}

// https://github.com/mark3labs/mcp-go/blob/main/examples/everything/main.go
func Serve(args []string) error {
	cfg, err := internal.ParseConfig(args)
	if err != nil {
		return err
	}

	mcpServer, err := NewMCPServer(cfg)
	if err != nil {
		return err
	}

	if config.Transport == "http" {
		addr := fmt.Sprintf("%s:%v", config.Host, config.Port)

		httpServer := server.NewStreamableHTTPServer(mcpServer)
		log.Infof("http server listening on :%s/mcp\n", addr)

		if err := httpServer.Start(addr); err != nil {
			return fmt.Errorf("http server error: %v", err)
		}
	} else {
		if err := server.ServeStdio(mcpServer); err != nil {
			return fmt.Errorf("stdio server error: %v", err)
		}
	}

	return nil
}

func NewMCPServer(cfg *api.AppConfig) (*server.MCPServer, error) {
	log.Debugf("config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.DBCred)

	vars, err := swarm.InitVars(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %v", err)
	}

	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		log.Debugf("beforeAny: %s, %v, %v\n", method, id, message)
	})
	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		log.Debugf("onSuccess: %s, %v, %v, %v\n", method, id, message, result)
	})
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		log.Debugf("onError: %s, %v, %v, %v\n", method, id, message, err)
	})
	hooks.AddBeforeInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest) {
		log.Debugf("beforeInitialize: %v, %v\n", id, message)
	})
	hooks.AddOnRequestInitialization(func(ctx context.Context, id any, message any) error {
		log.Debugf("AddOnRequestInitialization: %v, %v\n", id, message)
		// authorization verification and other preprocessing tasks are performed.
		return nil
	})
	hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		log.Debugf("afterInitialize: %v, %v, %v\n", id, message, result)
	})
	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		log.Debugf("afterCallTool: %v, %v, %v\n", id, message, result)
	})
	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		log.Debugf("beforeCallTool: %v, %v\n", id, message)
	})

	mcpServer := server.NewMCPServer(
		ServerName,
		ServerVersion,
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(false),
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	toolsMap := vars.ListTools()
	for i, v := range toolsMap {
		log.Debugf("tool [%v]: %s %v\n", i, v.ID(), v)

		if err := addTool(mcpServer, vars, v); err != nil {
			log.Infof("failed to add tool [%v]: %v\n", i, err)
		}
	}
	return mcpServer, nil
}

func addTool(server *server.MCPServer, vars *api.Vars, toolFunc *api.ToolFunc) error {
	toSchema := func(m map[string]any) mcp.ToolInputSchema {
		var s = mcp.ToolInputSchema{
			Type:       "object",
			Properties: make(map[string]any),
			Required:   nil,
		}
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

	// mcp.NewTool()
	tool := mcp.Tool{
		Name:        toolFunc.ID(),
		Description: toolFunc.Description,
		InputSchema: toSchema(toolFunc.Parameters),
		Annotations: mcp.ToolAnnotation{
			Title:           "",
			ReadOnlyHint:    mcp.ToBoolPtr(false),
			DestructiveHint: mcp.ToBoolPtr(true),
			IdempotentHint:  mcp.ToBoolPtr(false),
			OpenWorldHint:   mcp.ToBoolPtr(true),
		},
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

func addMcpFlags(cmd *cobra.Command) {
	var defaultPort = 5048
	if v := os.Getenv("AI_MCP_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &defaultPort)
	}
	var defaultHost = "localhost"
	if v := os.Getenv("AI_MCP_HOST"); v != "" {
		fmt.Sscanf(v, "%d", &defaultPort)
	}

	flags := cmd.Flags()

	// flags
	flags.IntVar(&config.Port, "port", defaultPort, "Port to run the server")
	flags.StringVar(&config.Host, "host", defaultHost, "Host to bind the server")

	flags.VarP(newTransportValue("http", &config.Transport), "transport", "t", "Transport protocol to use: http or stdio")

	// flags.StringVar(&config.McpServerUrl, "mcp-server-url", "", "MCP server URL")

	//
	// flags.String("log", "", "Log all debugging information to a file")
	// flags.Bool("verbose", false, "Show debugging information")
}
