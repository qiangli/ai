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
