package swarm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	ToolTypeShell    = "shell"
	ToolTypeSql      = "sql"
	ToolTypeMcp      = "mcp"
	ToolTypeAgent    = "agent"
	ToolTypeFunc     = "func"
)

func initTools(app *api.AppConfig) error {
	kits, err := LoadToolsConfig(app)
	if err != nil {
		log.Errorf("failed to load default tool config: %v\n", err)
		return err
	}

	app.ToolRegistry = make(map[string]*api.ToolFunc)

	conditionMet := func(name string, c *api.ToolCondition) bool {
		if c == nil {
			return true
		}
		if len(c.Env) > 0 {
			for _, v := range c.Env {
				if _, ok := os.LookupEnv(v); !ok {
					return false
				}
			}
		}
		if c.Lookup != nil {
			_, err := exec.LookPath(name)
			if err != nil {
				return false
			}
		}
		if len(c.Shell) > 0 {
			// get current shell name
			shellPath := os.Getenv("SHELL")
			shellName := filepath.Base(shellPath)
			_, ok := c.Shell[shellName]
			if !ok {
				return false
			}
		}
		return true
	}

	for _, config := range kits {
		for _, v := range config.Tools {
			log.Debugf("Kit: %s tool: %s - %s internal: %v\n", v.Kit, v.Name, v.Description, v.Internal)

			if v.Internal && !app.Internal {
				continue
			}

			// condition check
			if !conditionMet(v.Name, v.Condition) {
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

			// override
			app.ToolRegistry[tool.ID()] = tool

			// TODO this is used for security check by the evalCommand
			if v.Type == "system" {
				app.SystemTools = append(app.SystemTools, tool)
			}
		}

	}

	// //
	// // required system commands
	// commandMap := make(map[string]bool)
	// for _, config := range kits {
	// 	for _, v := range config.Commands {
	// 		commandMap[v] = true
	// 	}
	// }

	// app.ToolSystemCommands = make([]string, 0, len(commandMap))
	// for k := range commandMap {
	// 	app.ToolSystemCommands = append(app.ToolSystemCommands, k)
	// }

	return nil
}

func listDefaultTools(app *api.AppConfig) []*api.ToolFunc {
	var tools []*api.ToolFunc
	for _, v := range app.ToolRegistry {
		tools = append(tools, v)
	}
	return tools
}

func LoadToolsAsset(app *api.AppConfig, as api.AssetStore, base string, kits map[string]*api.ToolsConfig) error {
	dirs, err := as.ReadDir(base)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}
	for _, dir := range dirs {
		if dir.IsDir() {
			continue
		}
		f, err := as.ReadFile(filepath.Join(base, dir.Name()))
		if err != nil {
			return fmt.Errorf("failed to read tool file %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			continue
		}
		kit, err := LoadToolData(app, [][]byte{f})
		if err != nil {
			return err
		}
		kits[kit.Kit] = kit
	}

	return nil
}

func LoadToolsConfig(app *api.AppConfig) (map[string]*api.ToolsConfig, error) {
	var kits = make(map[string]*api.ToolsConfig)
	// default
	if err := LoadResourceToolsConfig(app, kits); err != nil {
		return nil, err
	}
	// web
	if err := LoadWebToolsConfig(app, kits); err != nil {
		log.Error("failed to load tools from web resource: %v\n", err)
	}
	// external/custom
	if err := LoadFileToolsConfig(app, kits); err != nil {
		log.Errorf("failed to load custom tools: %v\n", err)
	}
	return kits, nil
}

func LoadResourceToolsConfig(app *api.AppConfig, kits map[string]*api.ToolsConfig) error {
	rs := &ResourceStore{
		Base: "resource",
	}
	return LoadToolsAsset(app, rs, "tools", kits)
}

func LoadFileToolsConfig(app *api.AppConfig, kits map[string]*api.ToolsConfig) error {
	abs, err := filepath.Abs(app.Base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", app.Base, err)
	}
	fs := &FileStore{
		Base: abs,
	}
	return LoadToolsAsset(app, fs, "tools", kits)
}

func LoadWebToolsConfig(app *api.AppConfig, kits map[string]*api.ToolsConfig) error {
	if app.AgentResource == nil {
		return nil
	}
	for _, base := range app.AgentResource.Bases {
		ws := &WebStore{
			Base: base,
		}
		if err := LoadToolsAsset(app, ws, "tools", kits); err != nil {
			log.Errorf("failed to load tools from %q error: %v\n", base, err)
		}
	}
	return nil
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
			log.Debugf("Skipping internal tools: %v\n", tc)
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
	log.Infof("⣿ %s %+v\n", name, args)

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
		if v.State != api.StateDefault {
			return &api.Result{
				NextAgent: nextAgent,
				State:     v.State,
			}, nil
		}
		return callAgent(ctx, vars, nextAgent, args)
	case ToolTypeMcp:
		// spinner
		sp.Start()
		defer sp.Stop()

		out, err := callMcpTool(ctx, vars, name, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolTypeSystem:
		return callSystemTool(ctx, vars, v, args)
	case ToolTypeTemplate, ToolTypeShell, ToolTypeSql:
		out, err := callTplTool(ctx, vars, v, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolTypeFunc:
		// TODO
		if v.Name == "agent_transfer" {
			return callAgentTransfer(ctx, vars, v.Name, args)
		}
		out, err := callFuncTool(ctx, vars, v, args)
		return &api.Result{
			Value: out,
		}, err
	}

	return nil, fmt.Errorf("no such tool: %s", v.ID())
}

// func listAgentTools(app *api.AppConfig) ([]*api.ToolFunc, error) {
// 	tools := make([]*api.ToolFunc, 0)
// 	for _, v := range app.AgentToolMap {
// 		v.Type = "agent"
// 		tools = append(tools, v)
// 	}
// 	return tools, nil
// }

// listTools returns a list of all available tools, including agent, mcp, system, and function tools.
// This is for CLI
func listTools(app *api.AppConfig) ([]*api.ToolFunc, error) {
	list := []*api.ToolFunc{}

	// // agent tools
	// agentTools, err := listAgentTools(app)
	// if err != nil {
	// 	return nil, err
	// }
	// list = append(list, agentTools...)

	// mcp tools
	mcpTools, err := listMcpTools(app.McpServers)
	if err != nil {
		return nil, err
	}
	for _, v := range mcpTools {
		list = append(list, v)
	}

	// system and misc tools
	sysTools := listDefaultTools(app)
	list = append(list, sysTools...)

	return list, nil
}
