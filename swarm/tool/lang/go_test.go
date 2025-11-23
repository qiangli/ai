package lang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGoRunner(t *testing.T) {
	sourceCode, err := os.ReadFile("testdata/code.txt")
	if err != nil {
		t.Fatalf("Failed to read source code: %v", err)
	}

	dir, _ := filepath.Abs("testdata/")
	args := make(map[string]any)
	args["path"] = dir
	args["pattern"] = `*od*`
	runner := NewGoRunner("/tmp/gorun")
	sout, serr, err := runner.Run(string(sourceCode), "search", args)
	if err != nil {
		t.Errorf("GoRunner execution failed: %v", err)
	}

	t.Logf("stdout: %s", sout)
	t.Logf("stderr: %s", serr)
}
