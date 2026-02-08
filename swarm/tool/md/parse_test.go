package md

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

var update = flag.Bool("update", false, "update golden files")

func marshalTaskFileCanonical(tf *api.TaskFile) ([]byte, error) {
	m := map[string]any{}
	m["title"] = tf.Title
	m["description"] = tf.Description
	m["arguments"] = tf.Arguments
	groups := make([]string, 0, len(tf.Tasks))
	for k := range tf.Tasks {
		groups = append(groups, k)
	}
	sort.Strings(groups)
	tasksObj := map[string]any{}
	for _, g := range groups {
		tasksObj[g] = tf.Tasks[g]
	}
	m["Tasks"] = tasksObj
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func TestParseGolden(t *testing.T) {
	flag.Parse()
	fixturesDir := "./testdata/fixtures"
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Skip("no fixtures dir")
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		path := filepath.Join(fixturesDir, name)
		t.Run(name, func(t *testing.T) {
			c, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			expectErr := strings.HasSuffix(name, ".bad.md")
			tf, err := Parse(string(c))
			if expectErr {
				if err == nil {
					t.Fatalf("expected error parsing %s, got nil", name)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse error %s: %v", name, err)
			}
			b, err := marshalTaskFileCanonical(tf)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			goldenPath := filepath.Join(fixturesDir, name+".golden.json")
			if *update {
				os.WriteFile(goldenPath, b, 0o644)
				return
			}
			gb, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("no golden %s", goldenPath)
			}
			if string(gb) != string(b) {
				t.Errorf("golden mismatch %s", name)
			}
		})
	}
}

func TestParseBasic(t *testing.T) {
	c, err := os.ReadFile("./testdata/task.md")
	if err != nil {
		t.Fatal(err)
	}
	tf, err := Parse(string(c))
	if err != nil {
		t.Fatal(err)
	}
	if tf.Title == "" {
		t.Error("no title")
	}
	if tf.Arguments == "" {
		t.Error("no arguments")
	}
	if len(tf.Tasks) == 0 {
		t.Error("no tasks")
	}
	t.Run("TaskFile", func(t *testing.T) {
		if len(tf.Tasks) != 1 {
			t.Errorf("expected 1 group, got %d", len(tf.Tasks))
		}
		if _, ok := tf.Tasks["default"]; !ok {
			t.Error("missing default group")
		}
		hasYaml := false
		for _, task := range tf.Tasks["default"] {
			if task.MimeType == "yaml" {
				hasYaml = true
				break
			}
		}
		if !hasYaml {
			t.Error("no task with MimeType yaml in default")
		}
	})
}

func TestDebugDeps(t *testing.T) {
	source := `# Test
### Task Two
Second task

---
dependencies:
  - task-one
---

` + "```bash\necho two\n```"

	tf, err := Parse(source)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	tasks := tf.Tasks["default"]
	if len(tasks) == 0 {
		t.Fatal("no tasks")
	}

	task := tasks[0]
	t.Logf("Task: name=%s display=%s", task.Name, task.Display)
	t.Logf("Dependencies count: %d", len(task.Dependencies))
	for i, dep := range task.Dependencies {
		t.Logf("  Dep %d: %s (display: %s)", i, dep.Name, dep.Display)
	}
}

func TestDebugAll(t *testing.T) {
	c, err := os.ReadFile("./testdata/task.md")
	if err != nil {
		t.Fatal(err)
	}
	tf, err := Parse(string(c))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	defaultTasks := tf.Tasks["default"]
	taskMap := make(map[string]*api.Task)
	for _, task := range defaultTasks {
		taskMap[task.Name] = task
	}

	allTask, ok := taskMap["all"]
	if !ok {
		t.Fatal("all task not found")
	}

	t.Logf("All task: name=%s display=%s mime=%s", allTask.Name, allTask.Display, allTask.MimeType)
	t.Logf("All task description: %s", allTask.Description)
	t.Logf("All task dependencies count: %d", len(allTask.Dependencies))
	for i, dep := range allTask.Dependencies {
		t.Logf("  Dep %d: %s (display: %s)", i, dep.Name, dep.Display)
	}
	t.Logf("All task arguments: %v", allTask.Arguments)
	t.Logf("All task content length: %d", len(allTask.Content))
}

func TestParseTasks(t *testing.T) {
	c, err := os.ReadFile("./testdata/task.md")
	if err != nil {
		t.Fatal(err)
	}
	tf, err := Parse(string(c))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	defaultTasks := tf.Tasks["default"]
	if len(defaultTasks) == 0 {
		t.Fatal("no tasks in default group")
	}

	// Build task map for lookup
	taskMap := make(map[string]*api.Task)
	for _, task := range defaultTasks {
		taskMap[task.Name] = task
	}

	t.Run("DefaultTask", func(t *testing.T) {
		task, ok := taskMap["default"]
		if !ok {
			t.Fatal("default task not found")
		}
		if task.Display == "" {
			t.Error("default task has no display name")
		}
		if task.MimeType != "yaml" {
			t.Errorf("expected yaml mime type, got %q", task.MimeType)
		}
		if task.Content == "" {
			t.Error("default task has no content")
		}
	})

	t.Run("BuildTask", func(t *testing.T) {
		task, ok := taskMap["build"]
		if !ok {
			t.Fatal("build task not found")
		}
		if task.Display != "Build" {
			t.Errorf("expected display 'Build', got %q", task.Display)
		}
		if task.MimeType != "bash" {
			t.Errorf("expected bash mime type, got %q", task.MimeType)
		}
		if !strings.Contains(task.Content, "build.sh") {
			t.Error("build task content doesn't contain build.sh")
		}
	})

	t.Run("AllTaskWithDeps", func(t *testing.T) {
		task, ok := taskMap["all"]
		if !ok {
			t.Fatal("all task not found")
		}
		if len(task.Dependencies) == 0 {
			t.Error("all task has no dependencies")
		}
		// Check that dependencies are parsed
		expectedDeps := map[string]bool{"tidy": true, "build": true, "test": true, "install": true}
		for _, dep := range task.Dependencies {
			if !expectedDeps[dep.Name] {
				t.Errorf("unexpected dependency: %s", dep.Name)
			}
			delete(expectedDeps, dep.Name)
		}
		if len(expectedDeps) > 0 {
			t.Errorf("missing dependencies: %v", expectedDeps)
		}
	})

	t.Run("TestTask", func(t *testing.T) {
		task, ok := taskMap["test"]
		if !ok {
			t.Fatal("test task not found")
		}
		if task.MimeType != "bash" {
			t.Errorf("expected bash mime type, got %q", task.MimeType)
		}
		if !strings.Contains(task.Content, "go test") {
			t.Error("test task content doesn't contain 'go test'")
		}
	})
}
