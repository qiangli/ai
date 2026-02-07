package gitkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestStatus(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		dir := setupCleanRepo(t)
		out, _, err := Status(dir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if !strings.Contains(out, "On branch main") {
			t.Errorf("expected 'On branch main', got:\n%s", out)
		}
		if !strings.Contains(out, "nothing to commit, working tree clean") {
			t.Errorf("expected clean msg, got:\n%s", out)
		}
	})

	t.Run("unstaged_modified", func(t *testing.T) {
		dir := setupCleanRepo(t)
		readme := filepath.Join(dir, "README.md")
		content := []byte("# init\n\nmodified unstaged\n")
		if err := os.WriteFile(readme, content, 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		out, _, err := Status(dir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if !strings.Contains(out, "Changes not staged for commit:") {
			t.Errorf("expected unstaged section, got:\n%s", out)
		}
		if !strings.Contains(out, "  modified:   README.md") {
			t.Errorf("expected modified file, got:\n%s", out)
		}
	})

	t.Run("untracked", func(t *testing.T) {
		dir := setupCleanRepo(t)
		newfile := filepath.Join(dir, "untracked.txt")
		if err := os.WriteFile(newfile, []byte("untracked"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		out, _, err := Status(dir)
		if err != nil {
			t.Fatalf("Status failed: %v", err)
		}
		if !strings.Contains(out, "Untracked files:") {
			t.Errorf("expected untracked section, got:\n%s", out)
		}
		if !strings.Contains(out, "        untracked.txt") {
			t.Errorf("expected untracked.txt line, got:\n%s", out)
		}
	})

	t.Run("ahead", func(t *testing.T) {
		dir := setupCleanRepo(t)
		repo, err := git.PlainOpen(dir)
		if err != nil {
			t.Fatalf("PlainOpen: %v", err)
		}
		// Set branch config for upstream
		cfg, err := repo.Config()
		if err != nil {
			t.Fatalf("Config: %v", err)
		}
		// Add remote origin
		if cfg.Remotes == nil {
			cfg.Remotes = make(map[string]*config.RemoteConfig)
		}
		cfg.Remotes["origin"] = &config.RemoteConfig{
			Name:  "origin",
			URLs:  []string{"https://github.com/example/repo.git"},
			Fetch: []config.RefSpec{"+refs/heads/*:refs/remotes/origin/*"},
		}
		// Set branch.main upstream
		cfg.Branches = map[string]*config.Branch{
			"main": {
				Name:   "main",
				Remote: "origin",
				Merge:  plumbing.NewBranchReferenceName("main"),
			},
		}
		if err := repo.Storer.SetConfig(cfg); err != nil {
			t.Fatalf("SetConfig: %v", err)
		}
		// Set fake upstream ref to current HEAD (before new commit)
		head, err := repo.Head()
		if err != nil {
			t.Fatalf("Head: %v", err)
		}
		upRefName := plumbing.NewRemoteReferenceName("origin", "main")
		upRef := plumbing.NewHashReference(upRefName, head.Hash())
		if err := repo.Storer.SetReference(upRef); err != nil {
			t.Fatalf("SetReference: %v", err)
		}
		// Create local commit ahead
		wt, err := repo.Worktree()
		if err != nil {
			t.Fatalf("Worktree: %v", err)
		}
		localFile := filepath.Join(dir, "local.txt")
		if err := os.WriteFile(localFile, []byte("local commit"), 0o644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		if _, err := wt.Add("local.txt"); err != nil {
			t.Fatalf("Add: %v", err)
		}
		_, err = wt.Commit("local ahead commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test",
				Email: "test@test",
				When:  time.Now(),
			},
		})
		if err != nil {
			t.Fatalf("Commit: %v", err)
		}
		// Check status
		out, _, err := Status(dir)
		if err != nil {
			t.Fatalf("Status: %v", err)
		}
		if !strings.Contains(out, "Your branch is ahead of 'origin/main' by 1 commit.") {
			t.Errorf("expected ahead message, got:\n%s", out)
		}
		if !strings.Contains(out, `[use "git:push" to publish your local commits.]`) {
			t.Errorf("expected push hint, got:\n%s", out)
		}
	})
}

func setupCleanRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	fpath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(fpath, []byte("# hello\ngitkit test"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err = wt.Add("README.md"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	_, err = wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test",
			Email: "test@test",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
	// Rename master to main
	headRef, err := repo.Head()
	if err != nil {
		t.Fatalf("get Head: %v", err)
	}
	mainRefName := plumbing.NewBranchReferenceName("main")
	mainRef := plumbing.NewHashReference(mainRefName, headRef.Hash())
	if err := repo.Storer.SetReference(mainRef); err != nil {
		t.Fatalf("Set main ref: %v", err)
	}
	headSymRef := plumbing.NewSymbolicReference(plumbing.HEAD, mainRefName)
	if err := repo.Storer.SetReference(headSymRef); err != nil {
		t.Fatalf("Set HEAD symref: %v", err)
	}
	return dir
}
