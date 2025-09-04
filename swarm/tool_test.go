package swarm

import (
	"context"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestLoadToolsConfig(t *testing.T) {
	app := &api.AppConfig{
		Base: "../internal/data/",
	}
	kits, err := LoadToolsConfig(app)
	if err != nil {
		t.Fatalf("failed to load tool files: %v", err)
	}

	for _, config := range kits {
		for _, tool := range config.Tools {
			if tool.Name == "" {
				t.Fatal("tool name is empty")
			}
			if tool.Description == "" {
				t.Fatal("tool description is empty")
			}
			t.Logf("Kit: %s tool: %s - %s", tool.Kit, tool.Name, tool.Description)
		}
	}
}

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
			kit:  "host",
			name: "who_am_i",
			args: map[string]any{},
		},
		{
			kit:  "io",
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

	cfg := &api.AppConfig{
		Base:   "../internal/data/",
		Models: "openai",
	}

	ctx := context.Background()
	vars, err := InitVars(cfg)

	if err != nil {
		t.Fatalf("failed to initialize vars: %v", err)
	}

	ls := NewLocalSystem(cfg)

	for _, tt := range tests {
		tf := &api.ToolFunc{
			Kit:        tt.kit,
			Name:       tt.name,
			Parameters: map[string]any{},
		}
		result, err := ls.Call(ctx, vars, tf, tt.args)
		if err != nil {
			t.Fatalf("failed to call system tool: %v", err)
		}
		t.Logf("Result: %+v", result)
	}
}

// func TestCallDevTools(t *testing.T) {
// 	tests := []struct {
// 		id   string
// 		args map[string]any
// 	}{
// 		// {
// 		// 	id: "dev__git",
// 		// 	args: map[string]any{
// 		// 		"command": "status",
// 		// 	},
// 		// },
// 		// {
// 		// 	id: "dev__go",
// 		// 	args: map[string]any{
// 		// 		"command": "version",
// 		// 	},
// 		// },
// 		// {
// 		// 	id: "git__git",
// 		// 	args: map[string]any{
// 		// 		"command": "version",
// 		// 	},
// 		// },
// 	}

// 	cfg := &api.AppConfig{}

// 	ctx := context.Background()
// 	vars, _ := InitVars(cfg)

// 	for _, tt := range tests {
// 		// tf, ok := cfg.ToolRegistry[tt.id]
// 		// if !ok {
// 		// 	t.Fatalf("tool %s not found in registry", tt.id)
// 		// }
// 		tf, err := cfg.ToolLoader(tt.id)
// 		if err != nil {
// 			t.Fatalf("tool %s not found in registry", tt.id)
// 		}
// 		result, err := callTplTool(ctx, vars, tf[0], tt.args)
// 		if err != nil {
// 			t.Fatalf("failed to call dev tool %s: %v", tt.id, err)
// 		}
// 		t.Logf("Result for %s: %+v", tt.id, result)
// 	}
// }

// func TestCallFindTools(t *testing.T) {

// 	tests := []struct {
// 		id   string
// 		args map[string]any
// 	}{
// 		// {
// 		// 	id: "find__grep",
// 		// 	args: map[string]any{
// 		// 		"args": []string{"-rnw", "'pr_json_to_markdown'", "."},
// 		// 	},
// 		// },
// 	}

// 	cfg := &api.AppConfig{}

// 	ctx := context.Background()
// 	vars, _ := InitVars(cfg)

// 	for _, tt := range tests {
// 		// tf, ok := cfg.ToolRegistry[tt.id]
// 		// if !ok {
// 		// 	t.Fatalf("tool %s not found in registry", tt.id)
// 		// }
// 		tf, err := cfg.ToolLoader(tt.id)
// 		if err != nil {
// 			t.Fatalf("tool %s not found in registry", tt.id)
// 		}
// 		result, err := callTplTool(ctx, vars, tf[0], tt.args)
// 		if err != nil {
// 			t.Fatalf("failed to call dev tool %s: %v", tt.id, err)
// 		}
// 		t.Logf("Result for %s: %+v", tt.id, result)
// 	}
// }

// func TestCallShellTools(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}
// 	tests := []struct {
// 		id   string
// 		args map[string]any
// 	}{
// 		// {
// 		// 	id: "archive__zstd",
// 		// 	args: map[string]any{
// 		// 		"options": []string{"--verbose", "--force"},
// 		// 		"files":   []string{"test.zst"},
// 		// 	},
// 		// },
// 	}

// 	cfg := &api.AppConfig{}

// 	ctx := context.Background()
// 	vars, _ := InitVars(cfg)

// 	for _, tt := range tests {
// 		// tf, ok := cfg.ToolRegistry[tt.id]
// 		// if !ok {
// 		// 	t.Fatalf("tool %s not found in registry", tt.id)
// 		// }
// 		tf, err := cfg.ToolLoader(tt.id)
// 		if err != nil {
// 			t.Fatalf("tool %s not found in registry", tt.id)
// 		}
// 		result, err := callTplTool(ctx, vars, tf[0], tt.args)
// 		if err != nil {
// 			t.Fatalf("failed to call dev tool %s: %v", tt.id, err)
// 		}
// 		t.Logf("Result for %s: %+v", tt.id, result)
// 	}
// }

// func TestCallSqlTools(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}
// 	tests := []struct {
// 		id   string
// 		args map[string]any
// 	}{
// 		// {
// 		// 	id: "pg__db_query",
// 		// 	args: map[string]any{
// 		// 		"query": "SELECT version()",
// 		// 	},
// 		// },
// 	}

// 	cfg := &api.AppConfig{}

// 	cred := &api.DBCred{
// 		Host:     "localhost",
// 		Port:     "5432",
// 		Username: "",
// 		Password: "",
// 		DBName:   "postgres",
// 	}

// 	ctx := context.Background()
// 	vars, _ := InitVars(cfg)
// 	vars.Config.DBCred = cred

// 	for _, tt := range tests {
// 		// tf, ok := cfg.ToolRegistry[tt.id]
// 		// if !ok {
// 		// 	t.Fatalf("tool %s not found in registry", tt.id)
// 		// }
// 		tf, err := cfg.ToolLoader(tt.id)
// 		if err != nil {
// 			t.Fatalf("tool %s not found in registry", tt.id)
// 		}
// 		result, err := callTplTool(ctx, vars, tf[0], tt.args)
// 		if err != nil {
// 			t.Fatalf("failed to call dev tool %s: %v", tt.id, err)
// 		}
// 		t.Logf("Result for %s: %+v", tt.id, result)
// 	}
// }
