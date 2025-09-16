package swarm

import (
	"context"
	"embed"
	"fmt"
	"os"
	// "os/exec"
	"path/filepath"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/briandowns/spinner"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func ListTools(app *api.AppConfig) (map[string]*api.ToolFunc, error) {
	kits, err := LoadToolsConfig(app)
	if err != nil {
		log.Errorf("failed to load default tool config: %v\n", err)
		return nil, err
	}

	var toolRegistry = make(map[string]*api.ToolFunc)
	conditionMet := func(name string, c *api.ToolCondition) bool {
		// if c == nil {
		// 	return true
		// }
		// if len(c.Env) > 0 {
		// 	for _, k := range c.Env {
		// 		if v, ok := os.LookupEnv(k); !ok {
		// 			return false
		// 		} else {
		// 			app.Env[k] = v
		// 		}
		// 	}
		// }
		// if c.Lookup != nil {
		// 	_, err := exec.LookPath(name)
		// 	if err != nil {
		// 		return false
		// 	}
		// }
		// if len(c.Shell) > 0 {
		// 	// get current shell name
		// 	shellPath := os.Getenv("SHELL")
		// 	shellName := filepath.Base(shellPath)
		// 	_, ok := c.Shell[shellName]
		// 	if !ok {
		// 		return false
		// 	}
		// }
		return true
	}

	// in theory mcp and system/funcs type could be declared in the same kit
	for _, kit := range kits {
		// connector mcp
		if kit.Connector != nil {
			tools, err := ListMcpTools(kit)
			if err != nil {
				return nil, err
			}
			for _, tool := range tools {
				tool.Config = kit
				toolRegistry[tool.ID()] = tool
			}
		}

		// tools - inline
		if len(kit.Tools) > 0 {
			for _, v := range kit.Tools {
				log.Debugf("Kit: %s tool: %s - %s\n", kit.Kit, v.Name, v.Description)

				// condition check
				if !conditionMet(v.Name, v.Condition) {
					continue
				}
				tool := &api.ToolFunc{
					Type:        v.Type,
					Kit:         kit.Kit,
					Name:        v.Name,
					Description: v.Description,
					Parameters:  v.Parameters,
					Body:        v.Body,
					Config:      kit,
				}

				if tool.Type == "" {
					tool.Type = kit.Type
				}
				if tool.Type == "" {
					return nil, fmt.Errorf("Missing tool type: %s", v.Name)
				}

				// override
				toolRegistry[tool.ID()] = tool

				// TODO this is used for security check by the evalCommand
				if v.Type == api.ToolTypeSystem {
					app.SystemTools = append(app.SystemTools, tool)
				}
			}
		}
	}
	return toolRegistry, nil
}

func initTools(app *api.AppConfig) (func(string) ([]*api.ToolFunc, error), error) {
	tools, err := ListTools(app)
	if err != nil {
		return nil, err
	}

	getKit := func(kit string, name string) ([]*api.ToolFunc, error) {
		var list []*api.ToolFunc
		for _, v := range tools {
			if kit == "*" || kit == "" || v.Config.Kit == kit {
				if name == "*" || name == "" || v.Name == name {
					list = append(list, v)
				}
			}
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("no such tool kit. %s:%s", kit, name)
		}
		return list, nil
	}

	getType := func(toolType string, kit string) ([]*api.ToolFunc, error) {
		var list []*api.ToolFunc
		for _, v := range tools {
			if toolType == "*" || toolType == "" || v.Type == toolType {
				if kit == "*" || kit == "" || v.Config.Kit == kit {
					list = append(list, v)
				}
			}
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("no such tool: %s / %s", toolType, kit)
		}
		return list, nil
	}

	getTools := func(ns string, name string) ([]*api.ToolFunc, error) {
		if k, err := getKit(ns, name); err == nil {
			return k, nil
		}
		return getType(ns, name)
	}

	getTool := func(id string) ([]*api.ToolFunc, error) {
		if v, ok := tools[id]; ok {
			return []*api.ToolFunc{v}, nil
		}
		return nil, fmt.Errorf("no such tool: %s", id)
	}

	// type
	// kit:*
	// kit__name
	return func(s string) ([]*api.ToolFunc, error) {
		// id: kit__name
		if strings.Index(s, "__") > 0 {
			return getTool(s)
		}
		// type: type:kit
		// ktt: kit:name
		ns, n := split2(s, ":", "*")
		return getTools(ns, n)
	}, nil
}

func LoadToolsAsset(as api.AssetStore, base string, kits map[string]*api.ToolsConfig) error {
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
		kit, err := LoadToolData([][]byte{f})
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
	if err := LoadResourceToolsConfig(resourceFS, kits); err != nil {
		return nil, err
	}

	// external/custom
	if err := LoadFileToolsConfig(app.Base, kits); err != nil {
		log.Errorf("failed to load custom tools: %v\n", err)
	}

	if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
		if err := LoadWebToolsConfig(app.AgentResource.Resources, kits); err != nil {
			log.Errorf("failed to load tools from web resource: %v\n", err)
		}
	}

	return kits, nil
}

func LoadResourceToolsConfig(fs embed.FS, kits map[string]*api.ToolsConfig) error {
	rs := &ResourceStore{
		FS:   fs,
		Base: "resource",
	}
	return LoadToolsAsset(rs, "tools", kits)
}

func LoadFileToolsConfig(base string, kits map[string]*api.ToolsConfig) error {
	abs, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
	}
	// check if abs exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.Debugf("path does not exist: %s\n", abs)
		return nil
	}

	fs := &FileStore{
		Base: abs,
	}
	return LoadToolsAsset(fs, "tools", kits)
}

func LoadWebToolsConfig(resources []*api.Resource, kits map[string]*api.ToolsConfig) error {
	for _, v := range resources {
		ws := &WebStore{
			Base:  v.Base,
			Token: v.Token,
		}
		if err := LoadToolsAsset(ws, "tools", kits); err != nil {
			log.Errorf("failed to load tools from %q error: %v\n", v.Base, err)
		}
	}
	return nil
}

func LoadToolData(data [][]byte) (*api.ToolsConfig, error) {
	merged := &api.ToolsConfig{}

	for _, v := range data {
		tc := &api.ToolsConfig{}
		if err := yaml.Unmarshal(v, tc); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, tc, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	merged.Getenv = getenv

	return merged, nil
}

func NewToolCaller() api.ToolCaller {
	return func(vars *api.Vars, agent *api.Agent) func(context.Context, string, map[string]any) (*api.Result, error) {
		toolMap := make(map[string]*api.ToolFunc)
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}
		return func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
			log.Debugf("run tool: %s %+v\n", name, args)
			v, ok := toolMap[name]
			if !ok {
				return nil, fmt.Errorf("tool not found: %s", name)
			}
			return callTool(vars, v, ctx, name, args)
		}
	}
}

func callTool(vars *api.Vars, v *api.ToolFunc, ctx context.Context, name string, args map[string]any) (*api.Result, error) {
	log.Infof("⣿ %s %+v\n", name, args)

	result, err := dispatchTool(ctx, v, vars, name, args)

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

// dispatch tool by name (kit__name) and args
func dispatchTool(ctx context.Context, v *api.ToolFunc, vars *api.Vars, name string, args map[string]any) (*api.Result, error) {
	// spinner
	sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Suffix = " calling " + name + "\n"

	switch v.Type {
	// case ToolTypeAgent:
	// 	nextAgent := v.Kit
	// 	if v.Name != "" {
	// 		nextAgent = fmt.Sprintf("%s/%s", v.Kit, v.Name)
	// 	}
	// 	if v.State != api.StateDefault {
	// 		return &api.Result{
	// 			NextAgent: nextAgent,
	// 			State:     v.State,
	// 		}, nil
	// 	}
	// 	return callAgent(ctx, vars, nextAgent, args)
	case api.ToolTypeMcp:
		// spinner
		sp.Start()
		defer sp.Stop()

		out, err := callMcpTool(ctx, v, vars, name, args)
		return &api.Result{
			Value: out,
		}, err
	case api.ToolTypeSystem:
		local := newLocalSystem()
		return local.Call(ctx, vars, v, args)
	case api.ToolTypeWeb:
		out, err := callWebTool(ctx, vars, v, args)
		return &api.Result{
			Value: out,
		}, err
	case api.ToolTypeFunc:
		return callFuncTool(ctx, vars, v, args)
	}

	return nil, fmt.Errorf("no such tool: %s", name)
}
