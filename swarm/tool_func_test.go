package swarm

import (
	"context"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestCallSystemTool(t *testing.T) {
	tests := []struct {
		kit  string
		name string
		args map[string]any
	}{
		{
			kit:  "fs",
			name: "list_directory",
			args: map[string]any{"path": "/tmp"},
		},
		{
			kit:  "host",
			name: "workspace_dir",
			args: map[string]any{},
		},
		{
			kit:  "misc",
			name: "write_stdout",
			args: map[string]any{"content": "Hello, World!"},
		},
		{
			kit:  "os",
			name: "uname",
			args: map[string]any{},
		},
		{
			kit:  "os",
			name: "exec",
			args: map[string]any{
				"command": "which",
				"args":    []string{"go"},
			},
		},
	}

	cfg := &api.AppConfig{}

	ctx := context.Background()
	vars, _ := InitVars(cfg)

	for _, tt := range tests {
		tf := &api.ToolFunc{
			Kit:        tt.kit,
			Name:       tt.name,
			Parameters: map[string]any{},
		}
		result, err := callSystemTool(ctx, vars, tf, tt.args)
		if err != nil {
			t.Fatalf("failed to call system tool: %v", err)
		}
		t.Logf("Result: %+v", result)
	}
}

func TestCallDevTools(t *testing.T) {
	tests := []struct {
		id   string
		args map[string]any
	}{
		{
			id: "dev__git",
			args: map[string]any{
				"command": "status",
			},
		},
		{
			id: "dev__go",
			args: map[string]any{
				"command": "version",
			},
		},
		{
			id: "git__status",
			args: map[string]any{
				"command": "version",
			},
		},
		{
			id: "git__add",
			args: map[string]any{
				"files": []any{"pad", "poc/note.txt"},
			},
		},
	}

	cfg := &api.AppConfig{}
	initTools(cfg)

	ctx := context.Background()
	vars, _ := InitVars(cfg)

	for _, tt := range tests {
		tf, ok := toolRegistry[tt.id]
		if !ok {
			t.Fatalf("tool %s not found in registry", tt.id)
		}
		result, err := callTplTool(ctx, vars, tf, tt.args)
		if err != nil {
			t.Fatalf("failed to call dev tool %s: %v", tt.id, err)
		}
		t.Logf("Result for %s: %+v", tt.id, result)
	}
}
