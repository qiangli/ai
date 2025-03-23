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

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
)

type ToolConfig struct {
	Kit string `yaml:"kit"`

	Type        string         `yaml:"type"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Parameters  map[string]any `yaml:"parameters"`

	Body string `yaml:"body"`
}

type ToolsConfig struct {
	Kit   string        `yaml:"kit"`
	Tools []*ToolConfig `yaml:"tools"`
}

//go:embed resource/tools/*.yaml
var resourceTools embed.FS

func LoadDefaultToolConfig() (*ToolsConfig, error) {
	base := "resource/tools"
	dirs, err := resourceTools.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("failed to read testdata directory: %v", err)
	}
	var data [][]byte
	for _, dir := range dirs {
		if dir.IsDir() {
			continue
		}
		f, err := resourceTools.ReadFile(base + "/" + dir.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read tool file %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			continue
		}
		data = append(data, f)
	}
	return LoadToolData(data)
}

func LoadToolConfig(base string) (*ToolsConfig, error) {
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
	return LoadToolFiles(files)
}

func LoadToolFiles(file []string) (*ToolsConfig, error) {
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
	return LoadToolData(data)
}

func LoadToolData(data [][]byte) (*ToolsConfig, error) {
	merged := &ToolsConfig{}

	for _, v := range data {
		cfg := &ToolsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}
		// update kit if not set
		for _, tool := range cfg.Tools {
			if tool.Kit == "" {
				tool.Kit = cfg.Kit
			}
		}
		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

func LoadTools(config ToolsConfig) (map[string]*api.ToolFunc, error) {
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

func (r *Vars) CallTool(ctx context.Context, name string, args map[string]any) (*Result, error) {
	log.Infof("✨ %s %+v\n", name, args)

	result, err := dispatchTool(ctx, r, name, args)

	if err != nil {
		// log.Infof("❌ %s\n", err)
		log.Errorf("\033[31m✗\033[0m %s\n", err)
	} else {
		log.Infof("✔ %s\n", head(result.Value, 80))
	}

	return result, err
}

// head trims the string to the maxLen and replaces newlines with /.
func head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func dispatchTool(ctx context.Context, vars *Vars, name string, args map[string]any) (*Result, error) {

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
	case ToolTypeFunc:
		if fn, ok := vars.FuncRegistry[v.Name]; ok {
			return fn(ctx, vars, v.Name, args)
		}
	}

	return nil, fmt.Errorf("no such tool: %s", v.ID())
}
