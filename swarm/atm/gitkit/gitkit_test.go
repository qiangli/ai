package gitkit

import (
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

func TestExecGit_Version(t *testing.T) {
	if !hasGit(t) {
		return
	}
	stdout, stderr, err := ExecGit("", "--version")
	if err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(stdout, "git version") {
		t.Fatalf("expected 'git version' in stdout, got %q (stderr=%q)", stdout, stderr)
	}
}

func TestRunGitExitCode_NonZero(t *testing.T) {
	if !hasGit(t) {
		return
	}

	// Intentionally fail: unknown subcommand.
	_, _, code, err := RunGitExitCode("", "definitely-not-a-subcommand")
	if err == nil {
		t.Fatalf("expected error")
	}
	if code == 0 {
		t.Fatalf("expected non-zero exit code")
	}
}

func TestClone_LocalRepo_NoNetwork(t *testing.T) {
	if !hasGit(t) {
		return
	}

	tmp := t.TempDir()
	repoDir := filepath.Join(tmp, "repo")
	cloneDir := filepath.Join(tmp, "clone")

	initLocalRepo(t, repoDir)

	if err := Clone(repoDir, cloneDir); err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(cloneDir, "README.md"))
	if err != nil {
		t.Fatalf("read cloned file: %v", err)
	}
	if string(b) != "hello\n" {
		t.Fatalf("unexpected README.md content: %q", string(b))
	}
}

func TestCurrentBranch(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	branch, stderr, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch failed: %v (stderr=%q)", err, stderr)
	}
	// Default branch might be master or main depending on config; just ensure non-empty.
	if strings.TrimSpace(branch) == "" {
		t.Fatalf("expected non-empty branch")
	}
}

func TestListBranches(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	out, stderr, err := ListBranches(root)
	if err != nil {
		t.Fatalf("ListBranches failed: %v (stderr=%q)", err, stderr)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		t.Fatalf("expected at least one branch")
	}
}

func TestListRemotes(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	out, stderr, err := ListRemotes(root)
	if err != nil {
		t.Fatalf("ListRemotes failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "origin") {
		t.Fatalf("expected 'origin' in remotes output, got: %q", out)
	}
}

func TestLatestCommit(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	out, stderr, err := LatestCommit(root)
	if err != nil {
		t.Fatalf("LatestCommit failed: %v (stderr=%q)", err, stderr)
	}
	parts := strings.SplitN(strings.TrimSpace(out), " ", 2)
	if len(parts) != 2 {
		t.Fatalf("expected '<hash> <subject>', got %q", out)
	}
	if len(parts[0]) < 7 {
		t.Fatalf("expected hash prefix, got %q", parts[0])
	}
	if strings.TrimSpace(parts[1]) == "" {
		t.Fatalf("expected non-empty subject")
	}
}

func TestShowFileAtRev(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	out, stderr, err := ShowFileAtRev(root, "HEAD", "README.md")
	if err != nil {
		t.Fatalf("ShowFileAtRev failed: %v (stderr=%q)", err, stderr)
	}
	if out != "hello\n" {
		t.Fatalf("unexpected content: %q", out)
	}
}

func TestStatus(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create a new file
	if err := os.WriteFile(filepath.Join(root, "newfile.txt"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	out, stderr, err := Status(root)
	if err != nil {
		t.Fatalf("Status failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "newfile.txt") {
		t.Fatalf("expected 'newfile.txt' in status output, got: %q", out)
	}
}

func TestLog(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	logs, stderr, err := Log(root, 10, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("Log failed: %v (stderr=%q)", err, stderr)
	}
	if len(logs) == 0 {
		t.Fatalf("expected at least one commit in log")
	}
	if logs[0].Message != "init" {
		t.Fatalf("expected 'init' message, got %q", logs[0].Message)
	}
}

func TestCreateBranch(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	out, stderr, err := CreateBranch(root, "feature", "")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "feature") {
		t.Fatalf("expected 'feature' in output, got: %q", out)
	}

	// Verify the branch was created
	branches, _, err := ListBranches(root)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}
	if !strings.Contains(branches, "feature") {
		t.Fatalf("expected 'feature' branch in list, got: %q", branches)
	}
}

func TestCheckout(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create a new branch
	_, _, err := CreateBranch(root, "test-checkout", "")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	// Checkout the branch
	out, stderr, err := Checkout(root, "test-checkout")
	if err != nil {
		t.Fatalf("Checkout failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "test-checkout") {
		t.Fatalf("expected 'test-checkout' in output, got: %q", out)
	}

	// Verify we're on the new branch
	current, _, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch failed: %v", err)
	}
	if strings.TrimSpace(current) != "test-checkout" {
		t.Fatalf("expected current branch 'test-checkout', got: %q", current)
	}
}

func TestBranches(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create a test branch
	_, _, err := CreateBranch(root, "test-branch", "")
	if err != nil {
		t.Fatalf("CreateBranch failed: %v", err)
	}

	out, stderr, err := Branches(root, "local")
	if err != nil {
		t.Fatalf("Branches failed: %v (stderr=%q)", err, stderr)
	}

	// The output should be JSON array
	if !strings.HasPrefix(out, "[") || !strings.HasSuffix(out, "]") {
		t.Fatalf("expected JSON array output, got: %q", out)
	}
	if !strings.Contains(out, "test-branch") {
		t.Fatalf("expected 'test-branch' in branches, got: %q", out)
	}
}

func TestDiffUnstaged(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Modify a file
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("modified\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	out, stderr, err := DiffUnstaged(root, 3)
	if err != nil {
		t.Fatalf("DiffUnstaged failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "README.md") {
		t.Fatalf("expected 'README.md' in diff output, got: %q", out)
	}
}

func TestDiffStaged(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create and stage a new file
	if err := os.WriteFile(filepath.Join(root, "newfile.txt"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, _, err := Add(root, []string{"newfile.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	out, stderr, err := DiffStaged(root, 3)
	if err != nil {
		t.Fatalf("DiffStaged failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "newfile.txt") {
		t.Fatalf("expected 'newfile.txt' in diff output, got: %q", out)
	}
}

func TestDiffTarget(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create a second commit
	if err := os.WriteFile(filepath.Join(root, "file2.txt"), []byte("content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, _, err := Add(root, []string{"file2.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	repo, err := git.PlainOpen(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}
	_, err = wt.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "gitkit-test",
			Email: "gitkit-test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Diff against HEAD~1
	out, stderr, err := DiffTarget(root, "HEAD~1", 3)
	if err != nil {
		t.Fatalf("DiffTarget failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "file2.txt") {
		t.Fatalf("expected 'file2.txt' in diff output, got: %q", out)
	}
}

func TestReset(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Stage a file
	if err := os.WriteFile(filepath.Join(root, "staged.txt"), []byte("content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, _, err := Add(root, []string{"staged.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Reset
	out, stderr, err := Reset(root)
	if err != nil {
		t.Fatalf("Reset failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "unstaged") {
		t.Fatalf("expected 'unstaged' in output, got: %q", out)
	}

	// Verify file is no longer staged
	status, _, err := DiffStaged(root, 3)
	if err != nil {
		t.Fatalf("DiffStaged failed: %v", err)
	}
	if strings.Contains(status, "staged.txt") {
		t.Fatalf("expected file to be unstaged, but got: %q", status)
	}
}

func TestShow(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	out, stderr, err := Show(root, "HEAD")
	if err != nil {
		t.Fatalf("Show failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "commit") {
		t.Fatalf("expected 'commit' in output, got: %q", out)
	}
	if !strings.Contains(out, "init") {
		t.Fatalf("expected commit message 'init' in output, got: %q", out)
	}
}

func TestRevParse(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	hash, stderr, err := RevParse(root, "HEAD")
	if err != nil {
		t.Fatalf("RevParse failed: %v (stderr=%q)", err, stderr)
	}
	if len(hash) != 40 {
		t.Fatalf("expected 40-character hash, got: %q", hash)
	}
}

func TestAddMultipleFiles(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create multiple files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte("test\n"), 0o644); err != nil {
			t.Fatalf("write file %s: %v", f, err)
		}
	}

	// Add all files at once
	out, stderr, err := Add(root, files)
	if err != nil {
		t.Fatalf("Add failed: %v (stderr=%q)", err, stderr)
	}
	if !strings.Contains(out, "3") {
		t.Fatalf("expected '3' files added, got: %q", out)
	}
}

func TestLogWithTimeFilter(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Get current time
	now := time.Now()

	// Create a second commit
	if err := os.WriteFile(filepath.Join(root, "file2.txt"), []byte("content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, _, err := Add(root, []string{"file2.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	repo, err := git.PlainOpen(root)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("get worktree: %v", err)
	}
	_, err = wt.Commit("second commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "gitkit-test",
			Email: "gitkit-test@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("commit failed: %v", err)
	}

	// Test with time filter - should get both commits
	logs, _, err := Log(root, 10, now.Add(-1*time.Hour), now.Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(logs))
	}
}

func TestCommitWithAuthorInfo(t *testing.T) {
	if !hasGit(t) {
		return
	}

	root := filepath.Join(t.TempDir(), "repo")
	initLocalRepo(t, root)

	// Create and stage a file
	if err := os.WriteFile(filepath.Join(root, "test.txt"), []byte("content\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, _, err := Add(root, []string{"test.txt"}); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Commit with custom author
	args := []string{"-c", "user.name=Test Author", "-c", "user.email=test@example.com"}
	hash, stderr, code, err := Commit(root, "test commit", args)
	if err != nil {
		t.Fatalf("Commit failed: %v (stderr=%q, code=%d)", err, stderr, code)
	}
	if len(hash) != 40 {
		t.Fatalf("expected 40-character hash, got: %q", hash)
	}
}
