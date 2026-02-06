package gitkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

	if _, _, _, err := RunGitExitCode("", "init", root); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Create an origin remote to test ListRemotes deterministically.
	if _, _, _, err := RunGitExitCode(root, "remote", "add", "origin", "https://example.invalid/repo.git"); err != nil {
		t.Fatalf("git remote add failed: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	// if _, _, _, err := RunGitExitCode(root, "add", "README.md"); err != nil {
	// 	t.Fatalf("git add failed: %v", err)
	// }
	if _, _, err := Add(root, []string{"README.md"}); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Ensure commit can succeed in CI environments without global config.
	// if _, _, _, err := RunGitExitCode(root, "-c", "user.name=gitkit-test", "-c", "user.email=gitkit-test@example.com", "commit", "-m", "init"); err != nil {
	// 	t.Fatalf("git commit failed: %v", err)
	// }
	if _, _, _, err := Commit(root, "msg", []string{"-c", "user.name=gitkit-test", "-c", "user.email=gitkit-test@example.com", "-m", "init"}); err != nil {
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
