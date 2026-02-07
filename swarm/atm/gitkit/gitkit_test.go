package gitkit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func hasGit(t *testing.T) bool {
	t.Helper()
	_, _, _, err := RunGitExitCode("", "--version")
	if err != nil {
		t.Skipf("skipping: git is required but not available: %v", err)
		return false
	}
	return true
}

func initLocalRepo(t *testing.T, root string) {
	t.Helper()

	// Use go-git to initialize the repository
	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Create an origin remote
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.invalid/repo.git"},
	})
	if err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	// Create a test file
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Stage the file
	if _, _, err := Add(root, []string{"README.md"}); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Commit with explicit author info
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree failed: %v", err)
	}

	_, err = wt.Commit("init", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "gitkit-test",
			Email: "gitkit-test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("git commit failed: %v", err)
	}
}

func TestRunToolGitStatus(t *testing.T) {
	if !hasGit(t) {
		return
	}
	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	args := &Args{Dir: root}
	resAny, err := RunGitStatus(args)
	if err != nil {
		t.Fatalf("RunGitStatus failed: %v", err)
	}
	res, ok := resAny.(string)
	if !ok {
		t.Fatalf("result not string")
	}
	var out Output
	if err := json.Unmarshal([]byte(res), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !out.OK || out.ExitCode != 0 {
		t.Fatalf("bad result: exit=%d error=%q stdout=%q", out.ExitCode, out.Error, out.Stdout)
	}
	if strings.TrimSpace(out.Stdout) == "" {
		t.Fatal("empty stdout")
	}
}

func TestRunToolGitLog(t *testing.T) {
	if !hasGit(t) {
		return
	}
	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	args := &Args{Dir: root}
	resAny, err := RunGitLog(args)
	if err != nil {
		t.Fatalf("RunGitLog failed: %v", err)
	}
	res, ok := resAny.(string)
	if !ok {
		t.Fatalf("result not string")
	}
	var out Output
	if err := json.Unmarshal([]byte(res), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !out.OK || out.ExitCode != 0 {
		t.Fatalf("bad result: exit=%d error=%q stdout=%q", out.ExitCode, out.Error, out.Stdout)
	}
	if !strings.HasPrefix(out.Stdout, "[") || !strings.HasSuffix(out.Stdout, "]") {
		t.Fatalf("stdout not JSON array: %q", out.Stdout)
	}
}

func TestRunToolGitBranches(t *testing.T) {
	if !hasGit(t) {
		return
	}
	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	args := &Args{Dir: root}
	resAny, err := RunGitBranches(args)
	if err != nil {
		t.Fatalf("RunGitBranches failed: %v", err)
	}
	res, ok := resAny.(string)
	if !ok {
		t.Fatalf("result not string")
	}
	var out Output
	if err := json.Unmarshal([]byte(res), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !out.OK || out.ExitCode != 0 {
		t.Fatalf("bad result: exit=%d error=%q stdout=%q", out.ExitCode, out.Error, out.Stdout)
	}
	if !strings.HasPrefix(out.Stdout, "[") || !strings.HasSuffix(out.Stdout, "]") {
		t.Fatalf("stdout not JSON array: %q", out.Stdout)
	}
}

func TestRunToolGitAdd(t *testing.T) {
	if !hasGit(t) {
		return
	}
	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	newfile := filepath.Join(root, "newfile.txt")
	if err := os.WriteFile(newfile, []byte("test content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	args := &Args{Dir: root, Files: []string{"newfile.txt"}}
	resAny, err := RunGitAdd(args)
	if err != nil {
		t.Fatalf("RunGitAdd failed: %v", err)
	}
	res, ok := resAny.(string)
	if !ok {
		t.Fatalf("result not string")
	}
	var out Output
	if err := json.Unmarshal([]byte(res), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !out.OK || out.ExitCode != 0 {
		t.Fatalf("bad result: exit=%d error=%q stdout=%q", out.ExitCode, out.Error, out.Stdout)
	}
	if !strings.Contains(out.Stdout, "added") {
		t.Fatalf("expected 'added' in stdout: %q", out.Stdout)
	}
}
