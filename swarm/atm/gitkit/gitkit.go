package gitkit

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ExecGit executes the system 'git' binary with the provided arguments.
//
// Security: this function does not invoke a shell; it uses exec.Command with an
// explicit argv list, preventing shell interpolation.
//
// Args are passed exactly as provided. If dir is non-empty it is used as the
// working directory.
//
// Note: if you need the underlying git exit code, prefer RunGitExitCode.
func ExecGit(dir string, args ...string) (stdout string, stderr string, err error) {
	stdout, stderr, _, err = RunGitExitCode(dir, args...)
	if err != nil {
		// Keep compatibility: wrap with a bit more context.
		return stdout, stderr, fmt.Errorf("git %v failed: %w", args, err)
	}
	return stdout, stderr, nil
}

// RunGitExitCode executes the system 'git' binary with the provided arguments
// and returns stdout, stderr, and the process exit code.
//
// It does not use a shell.
//
// The returned err is the original cmd.Run() error (possibly *exec.ExitError),
// enabling callers to inspect it if needed.
func RunGitExitCode(dir string, args ...string) (stdout string, stderr string, exitCode int, err error) {
	gitPath, lookErr := exec.LookPath("git")
	if lookErr != nil {
		return "", "", 127, fmt.Errorf("git not found on PATH: %w", lookErr)
	}

	cmd := exec.Command(gitPath, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err == nil {
		return stdout, stderr, 0, nil
	}

	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return stdout, stderr, ee.ExitCode(), err
	}
	return stdout, stderr, 1, err
}

// Clone clones repoURL into destDir using `git clone`.
func Clone(repoURL, destDir string) error {
	_, stderr, _, err := RunGitExitCode("", "clone", repoURL, destDir)
	if err != nil {
		if stderr != "" {
			return fmt.Errorf("git clone failed: %w; stderr: %s", err, strings.TrimSpace(stderr))
		}
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

// Status returns `git status --porcelain=v1` output for the given directory.
func Status(dir string) (string, string, error) {
	return ExecGit(dir, "status", "--porcelain=v1")
}

// Commit performs `git commit -m <message>` in dir.
func Commit(dir string, message string) (string, string, error) {
	if strings.TrimSpace(message) == "" {
		return "", "", errors.New("commit message must not be empty")
	}
	return ExecGit(dir, "commit", "-m", message)
}

// Pull performs `git pull` (plus any extra args) in dir.
func Pull(dir string, args ...string) (string, string, error) {
	argv := append([]string{"pull"}, args...)
	return ExecGit(dir, argv...)
}

// Push performs `git push` (plus any extra args) in dir.
func Push(dir string, args ...string) (string, string, error) {
	argv := append([]string{"push"}, args...)
	return ExecGit(dir, argv...)
}

// CurrentBranch returns the current branch short name.
func CurrentBranch(dir string) (string, string, error) {
	out, errOut, err := ExecGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimSpace(out), errOut, err
}

// RemoteURL returns the URL for the default remote "origin".
func RemoteURL(dir string) (string, string, error) {
	out, errOut, err := ExecGit(dir, "remote", "get-url", "origin")
	return strings.TrimSpace(out), errOut, err
}

// RevParse resolves a revision to a commit hash (or other ref depending on args).
func RevParse(dir, rev string) (string, string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", "", errors.New("rev must not be empty")
	}
	out, errOut, err := ExecGit(dir, "rev-parse", rev)
	return strings.TrimSpace(out), errOut, err
}

// ListBranches lists local branch names, one per line.
//
// Uses: git branch --list --format=%(refname:short)
func ListBranches(dir string) (string, string, error) {
	out, errOut, _, err := RunGitExitCode(dir, "branch", "--list", "--format=%(refname:short)")
	return out, errOut, err
}

// ListRemotes lists remotes.
//
// Uses: git remote -v
func ListRemotes(dir string) (string, string, error) {
	out, errOut, _, err := RunGitExitCode(dir, "remote", "-v")
	return out, errOut, err
}

// LatestCommit returns the latest commit as a single line: "<hash> <subject>".
//
// Uses: git log -1 --pretty=format:%H %s
func LatestCommit(dir string) (string, string, error) {
	out, errOut, _, err := RunGitExitCode(dir, "log", "-1", "--pretty=format:%H %s")
	return strings.TrimSpace(out), errOut, err
}

// ShowFileAtRev returns the file content at the given revision.
//
// Uses: git show <rev>:<path>
func ShowFileAtRev(dir, rev, path string) (string, string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", "", errors.New("rev must not be empty")
	}
	if strings.TrimSpace(path) == "" {
		return "", "", errors.New("path must not be empty")
	}
	spec := fmt.Sprintf("%s:%s", rev, path)
	out, errOut, _, err := RunGitExitCode(dir, "show", spec)
	return out, errOut, err
}
