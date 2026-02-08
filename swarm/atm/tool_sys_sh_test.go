package atm

import (
	"context"
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/shell/vfs"
)

func TestRunTask(t *testing.T) {
	// Simple taskfile for testing
	taskfileContent := `# Test Tasks

## Tasks

### Echo Hello

Simple echo task

` + "```bash\necho 'Hello from task'\n```" + `

### Echo World

Another echo task

` + "```bash\necho 'World from task'\n```" + `

### Combined

Task with dependencies

---
dependencies:
  - echo-hello
  - echo-world
---

` + "```bash\necho 'Combined task'\n```"

	tests := []struct {
		name          string
		taskfile      string
		tasks         []string
		expectError   bool
		errorContains string
	}{
		{
			name:     "single task",
			taskfile: "data:," + taskfileContent,
			tasks:    []string{"echo-hello"},
		},
		{
			name:     "multiple tasks",
			taskfile: "data:," + taskfileContent,
			tasks:    []string{"echo-hello", "echo-world"},
		},
		{
			name:     "task with dependencies",
			taskfile: "data:," + taskfileContent,
			tasks:    []string{"combined"},
		},
		{
			name:        "no tasks specified",
			taskfile:    "data:," + taskfileContent,
			tasks:       []string{},
			expectError: false, // Should return message, not error
		},
		{
			name:          "nonexistent task",
			taskfile:      "data:," + taskfileContent,
			tasks:         []string{"nonexistent"},
			expectError:   true,
			errorContains: "not found",
		},
		{
			name:          "invalid taskfile",
			taskfile:      "data:,invalid content",
			tasks:         []string{"test"},
			expectError:   true,
			errorContains: "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal context for testing
			kit := &SystemKit{}
			vars := &api.Vars{
				Workspace: &testWorkspace{},
				RootAgent: &api.Agent{
					Runner: &testRunner{},
				},
			}
			args := map[string]any{
				"taskfile": tt.taskfile,
				"tasks":    tt.tasks,
			}

			result, err := kit.RunTask(context.Background(), vars, "run_task", args)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

// testWorkspace is a minimal workspace for testing (only supports data: URIs via LoadURIContent)
type testWorkspace struct{}

func (w *testWorkspace) ReadFile(name string, opts *vfs.ReadOptions) ([]byte, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) WriteFile(name string, data []byte) error {
	return fs.ErrPermission
}

func (w *testWorkspace) Locator(name string) (string, error) {
	return "", fs.ErrNotExist
}

func (w *testWorkspace) ListRoots() ([]string, error) {
	return []string{"/"}, nil
}

func (w *testWorkspace) ListDirectory(name string) ([]string, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) CreateDirectory(name string) error {
	return fs.ErrPermission
}

func (w *testWorkspace) MoveFile(oldName, newName string) error {
	return fs.ErrPermission
}

func (w *testWorkspace) GetFileInfo(name string) (*vfs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) DeleteFile(name string, recursive bool) error {
	return fs.ErrPermission
}

func (w *testWorkspace) CopyFile(src, dst string) error {
	return fs.ErrPermission
}

func (w *testWorkspace) EditFile(name string, opts *vfs.EditOptions) (int, error) {
	return 0, fs.ErrPermission
}

func (w *testWorkspace) Tree(path string, depth int, followSymlinks bool) (string, error) {
	return "", nil
}

func (w *testWorkspace) SearchFiles(pattern string, opts *vfs.SearchOptions) (string, error) {
	return "", nil
}

func (w *testWorkspace) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) ReadMultipleFiles(paths []string) ([]string, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) Lstat(name string) (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func (w *testWorkspace) Stat(name string) (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

// testRunner is a minimal runner for testing that just returns success
type testRunner struct{}

func (r *testRunner) Run(ctx context.Context, command string, args map[string]any) (any, error) {
	// For bash commands, just return success
	if command == "sh:bash" {
		script, _ := args["script"].(string)
		return "Executed: " + script, nil
	}
	return "Success", nil
}
