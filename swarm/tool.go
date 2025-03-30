package swarm

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/briandowns/spinner"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

const (
	ToolTypeSystem   = "system"
	ToolTypeTemplate = "template"
	ToolTypeMcp      = "mcp"
	ToolTypeAgent    = "agent"
	ToolTypeFunc     = "func"
)

var toolRegistry map[string]*api.ToolFunc
var toolSystemCommands []string

var systemTools []*api.ToolFunc

func initTools(app *api.AppConfig) error {
	config, err := LoadDefaultToolConfig(app)
	if err != nil {
		log.Errorf("failed to load default tool config: %v", err)
		return err
	}

	toolRegistry = make(map[string]*api.ToolFunc)
	for _, v := range config.Tools {

		log.Debugf("Kit: %s tool: %s - %s internal: %v", v.Kit, v.Name, v.Description, v.Internal)
		if v.Internal && !app.Internal {
			continue
		}

		tool := &api.ToolFunc{
			Type:        v.Type,
			Kit:         v.Kit,
			Name:        v.Name,
			Description: v.Description,
			Parameters:  v.Parameters,
			Body:        v.Body,
		}
		toolRegistry[tool.ID()] = tool

		// TODO this is used for security check by the evalCommand
		if v.Type == "system" {
			systemTools = append(systemTools, tool)
		}
	}

	// required system commands
	commandMap := make(map[string]bool)
	for _, v := range config.Commands {
		commandMap[v] = true
	}
	toolSystemCommands = make([]string, 0, len(commandMap))
	for k := range commandMap {
		toolSystemCommands = append(toolSystemCommands, k)
	}

	return nil
}

func listDefaultTools() []*api.ToolFunc {
	var tools []*api.ToolFunc
	for _, v := range toolRegistry {
		tools = append(tools, v)
	}
	return tools
}

// type ToolConfig struct {
// 	Kit string `yaml:"kit"`

// 	Type        string         `yaml:"type"`
// 	Name        string         `yaml:"name"`
// 	Description string         `yaml:"description"`
// 	Parameters  map[string]any `yaml:"parameters"`

// 	Body string `yaml:"body"`

// 	Internal bool `yaml:"internal"`
// }

// type ToolsConfig struct {
// 	Kit      string `yaml:"kit"`
// 	Internal bool   `yaml:"internal"`

// 	// system commands used by tools
// 	Commands []string `yaml:"commands"`

// 	Tools []*ToolConfig `yaml:"tools"`
// }

//go:embed resource/tools/*.yaml
var resourceTools embed.FS

func LoadDefaultToolConfig(app *api.AppConfig) (*api.ToolsConfig, error) {
	const base = "resource/tools"

	dirs, err := resourceTools.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata directory: %v", err)
	}
	var data [][]byte
	for _, dir := range dirs {
		if dir.IsDir() {
			continue
		}
		f, err := resourceTools.ReadFile(filepath.Join(base, dir.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read tool file %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			continue
		}
		data = append(data, f)
	}
	return LoadToolData(app, data)
}

func LoadToolConfig(app *api.AppConfig, base string) (*api.ToolsConfig, error) {
	dirs, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata directory: %v", err)
	}

	var files []string
	for _, dir := range dirs {
		if dir.IsDir() {
			continue
		}
		files = append(files, filepath.Join(base, dir.Name()))
	}
	return LoadToolFiles(app, files)
}

func LoadToolFiles(app *api.AppConfig, file []string) (*api.ToolsConfig, error) {
	var data [][]byte
	for _, f := range file {
		d, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read tool file %s: %w", f, err)
		}
		if len(d) == 0 {
			continue
		}
		data = append(data, d)
	}
	return LoadToolData(app, data)
}

func LoadToolData(app *api.AppConfig, data [][]byte) (*api.ToolsConfig, error) {
	merged := &api.ToolsConfig{}

	for _, v := range data {
		tc := &api.ToolsConfig{}
		if err := yaml.Unmarshal(v, tc); err != nil {
			return nil, err
		}
		// skip internal tools
		if tc.Internal && !app.Internal {
			log.Debugf("Skipping internal tools: %v", tc)
			continue
		}
		// update kit if not set
		for _, tool := range tc.Tools {
			if tool.Kit == "" {
				tool.Kit = tc.Kit
			}
		}
		if err := mergo.Merge(merged, tc, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func LoadTools(config api.ToolsConfig) (map[string]*api.ToolFunc, error) {
	toolRegistry := make(map[string]*api.ToolFunc)
	for _, toolConfig := range config.Tools {
		tool := &api.ToolFunc{
			Name:        toolConfig.Name,
			Description: toolConfig.Description,
			Parameters:  toolConfig.Parameters,
			Type:        toolConfig.Type,
		}
		if _, exists := toolRegistry[tool.Name]; exists {
			return nil, fmt.Errorf("duplicate tool name: %s", tool.Name)
		}
		toolRegistry[tool.Name] = tool
	}
	return toolRegistry, nil
}

func CallTool(ctx context.Context, vars *api.Vars, name string, args map[string]any) (*api.Result, error) {
	log.Infof("⚡ %s %+v\n", name, args)

	result, err := dispatchTool(ctx, vars, name, args)

	if err != nil {
		log.Errorf("\033[31m✗\033[0m %s\n", err)
	} else {
		log.Infof("✔ %s \n", head(result.String(), 180))
	}

	return result, err
}

// head trims the string to the maxLen and replaces newlines with /.
func head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "/")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func dispatchTool(ctx context.Context, vars *api.Vars, name string, args map[string]any) (*api.Result, error) {

	v, ok := vars.ToolRegistry[name]
	if !ok {
		return nil, fmt.Errorf("no such tool: %s", name)
	}

	// spinner
	sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Suffix = " calling " + name + "\n"

	switch v.Type {
	case ToolTypeAgent:
		nextAgent := v.Kit
		if v.Name != "" {
			nextAgent = fmt.Sprintf("%s/%s", v.Kit, v.Name)
		}
		return &api.Result{
			NextAgent: nextAgent,
			State:     api.StateTransfer,
		}, nil
	case ToolTypeMcp:
		// spinner
		sp.Start()
		defer sp.Stop()

		out, err := callMcpTool(ctx, vars, name, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolTypeSystem:
		out, err := callSystemTool(ctx, vars, v, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolTypeTemplate:
		out, err := callTplTool(ctx, vars, v, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolTypeFunc:
		if fn, ok := vars.FuncRegistry[v.Name]; ok {
			return fn(ctx, vars, v.Name, args)
		}
		out, err := callFuncTool(ctx, vars, v, args)
		return &api.Result{
			Value: out,
		}, err
	}

	return nil, fmt.Errorf("no such tool: %s", v.ID())
}

func listAgentTools() ([]*api.ToolFunc, error) {
	tools := make([]*api.ToolFunc, 0)
	for _, v := range agentToolMap {
		v.Type = "agent"
		tools = append(tools, v)
	}
	return tools, nil
}

// listTools returns a list of all available tools, including agent, mcp, system, and function tools.
// This is for CLI
func listTools(app *api.AppConfig) ([]*api.ToolFunc, error) {
	list := []*api.ToolFunc{}

	// agent tools
	agentTools, err := listAgentTools()
	if err != nil {
		return nil, err
	}
	list = append(list, agentTools...)

	// mcp tools
	mcpTools, err := listMcpTools(app.McpServerUrl)
	if err != nil {
		return nil, err
	}
	for _, v := range mcpTools {
		list = append(list, v...)
	}

	// system and misc tools
	sysTools := listDefaultTools()
	list = append(list, sysTools...)

	// function tools
	funcTools, err := ListFuncTools()
	if err != nil {
		return nil, err
	}
	list = append(list, funcTools...)

	return list, nil
}

// // ListServiceTools returns a list of all available tools for exporting (mcp and system tools).
// func ListServiceTools(mcpServerUrl string) ([]*api.ToolFunc, error) {
// 	list := []*api.ToolFunc{}

// 	// mcp tools
// 	mcpTools, err := listMcpTools(mcpServerUrl)
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, v := range mcpTools {
// 		list = append(list, v...)
// 	}

// 	// system and misc tools
// 	sysTools := listDefaultTools()
// 	list = append(list, sysTools...)

// 	return list, nil
// }
