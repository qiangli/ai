package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestDryRunBlocksDestructive(t *testing.T) {
	cmd := exec.Command("/usr/bin/env", "go", "run", "sh_jq.go", "-e", "del(.foo)")
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error when running destructive expression without --confirm; output: %s", string(out))
	}
	if !strings.Contains(string(out), "Destructive expression detected") {
		t.Fatalf("unexpected output: %s", string(out))
	}
}

func TestNonDestructiveWorks(t *testing.T) {
	cmd := exec.Command("/usr/bin/env", "go", "run", "sh_jq.go", "-e", ".foo")
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader("{\"foo\": 1}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("non-destructive should run: %v, output: %s", err, string(out))
	}
	if !strings.Contains(string(out), "1") {
		t.Fatalf("unexpected output: %s", string(out))
	}
}
