package gitkit

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ExecGit executes a supported git operation implemented in pure Go (go-git).
// It preserves the old signature: (stdout, stderr string, err error).
// Note: not all arbitrary git subcommands are supported. For unsupported
// commands ExecGit/RunGitExitCode will return a descriptive error.
func ExecGit(dir string, args ...string) (stdout string, stderr string, err error) {
	out, errOut, _, runErr := RunGitExitCode(dir, args...)
	if runErr != nil {
		return out, errOut, fmt.Errorf("git %v failed: %w", args, runErr)
	}
	return out, errOut, nil
}

// RunGitExitCode provides a minimal emulation of some git subcommands using
// go-git. It returns stdout, stderr, exitCode, err. For unsupported commands
// it returns exitCode 127 and an error.
func RunGitExitCode(dir string, args ...string) (stdout string, stderr string, exitCode int, err error) {
	if len(args) == 0 {
		return "", "", 2, fmt.Errorf("no git command provided")
	}

	cmd := args[0]
	switch cmd {
	case "--version":
		return "git version go-git (pure Go)\n", "", 0, nil
	case "init":
		// git init [path]
		path := dir
		if len(args) >= 2 && dir == "" {
			path = args[1]
		}
		if path == "" {
			path = "."
		}
		_, err := git.PlainInit(path, false)
		if err != nil {
			return "", err.Error(), 1, err
		}
		return "", "", 0, nil
	case "remote":
		if len(args) >= 2 && args[1] == "add" {
			// git remote add <name> <url>
			if len(args) < 4 {
				return "", "", 2, fmt.Errorf("remote add requires name and url")
			}
			name := args[2]
			url := args[3]
			repo, openErr := git.PlainOpen(dir)
			if openErr != nil {
				return "", openErr.Error(), 1, openErr
			}
			_, rerr := repo.CreateRemote(&config.RemoteConfig{Name: name, URLs: []string{url}})
			if rerr != nil {
				return "", rerr.Error(), 1, rerr
			}
			return "", "", 0, nil
		}
		return "", "", 127, fmt.Errorf("unsupported remote subcommand: %v", args[1:])
	case "add":
		if len(args) < 2 {
			return "", "", 2, fmt.Errorf("git add requires a path")
		}
		repo, openErr := git.PlainOpen(dir)
		if openErr != nil {
			return "", openErr.Error(), 1, openErr
		}
		wt, wtErr := repo.Worktree()
		if wtErr != nil {
			return "", wtErr.Error(), 1, wtErr
		}
		for _, p := range args[1:] {
			if _, aerr := wt.Add(p); aerr != nil {
				return "", aerr.Error(), 1, aerr
			}
		}
		return "", "", 0, nil
	case "commit":
		// Supports optional leading -c user.name=.. -c user.email=.. in args
		// and commit -m "msg"
		name := ""
		email := ""
		filtered := []string{}
		for i := 0; i < len(args); i++ {
			if args[i] == "-c" && i+1 < len(args) {
				kv := args[i+1]
				if strings.HasPrefix(kv, "user.name=") {
					name = strings.TrimPrefix(kv, "user.name=")
				} else if strings.HasPrefix(kv, "user.email=") {
					email = strings.TrimPrefix(kv, "user.email=")
				}
				i++
				continue
			}
			filtered = append(filtered, args[i])
		}
		msg := ""
		for i := 0; i < len(filtered); i++ {
			if filtered[i] == "-m" && i+1 < len(filtered) {
				msg = filtered[i+1]
				break
			}
		}
		if strings.TrimSpace(msg) == "" {
			return "", "", 2, fmt.Errorf("commit message must not be empty")
		}
		repo, openErr := git.PlainOpen(dir)
		if openErr != nil {
			return "", openErr.Error(), 1, openErr
		}
		wt, wtErr := repo.Worktree()
		if wtErr != nil {
			return "", wtErr.Error(), 1, wtErr
		}
		// Ensure index/staging from worktree is up-to-date; go-git requires Add prior to Commit
		// Assume caller ran 'add'. Proceed to commit.
		if name == "" || email == "" {
			// fall back to environment or generic
			if name == "" {
				if v := os.Getenv("GIT_AUTHOR_NAME"); v != "" {
					name = v
				}
			}
			if email == "" {
				if v := os.Getenv("GIT_AUTHOR_EMAIL"); v != "" {
					email = v
				}
			}
			if name == "" {
				name = "gitkit"
			}
			if email == "" {
				email = "gitkit@example.invalid"
			}
		}
		commitHash, cerr := wt.Commit(msg, &git.CommitOptions{
			Author: &object.Signature{Name: name, Email: email, When: time.Now()},
		})
		if cerr != nil {
			return "", cerr.Error(), 1, cerr
		}
		// Return the commit hash on stdout for convenience.
		return commitHash.String(), "", 0, nil
	case "-c":
		// if caller passed config flags then next arg may be actual command;
		// briefly handle when they call like: -c ... commit ...
		// Recurse after stripping leading -c pairs.
		filtered := []string{}
		for i := 0; i < len(args); i++ {
			if args[i] == "-c" && i+1 < len(args) {
				i++
				continue
			}
			filtered = append(filtered, args[i])
		}
		return RunGitExitCode(dir, filtered...)
	default:
		return "", "", 127, fmt.Errorf("unsupported git subcommand: %s", cmd)
	}
}

// Clone clones repoURL into destDir using go-git.
func Clone(repoURL, destDir string) error {
	_, err := git.PlainClone(destDir, false, &git.CloneOptions{URL: repoURL})
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

// Status returns a simplified status: list of changed files. It delegates to ExecGit
// for compatibility but provides a basic implementation.
func Status(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	st, err := wt.Status()
	if err != nil {
		return "", err.Error(), err
	}
	// Status.String returns a porcelain-like listing; reuse it.
	return st.String(), "", nil
}

func Add(dir string, args []string) (string, string, error) {
	repo, openErr := git.PlainOpen(dir)
	if openErr != nil {
		return "", openErr.Error(), openErr
	}
	wt, wtErr := repo.Worktree()
	if wtErr != nil {
		return "", wtErr.Error(), wtErr
	}
	for _, p := range args[1:] {
		if _, aerr := wt.Add(p); aerr != nil {
			return "", aerr.Error(), aerr
		}
	}
	return "", "", nil
}

// // Commit performs a commit in dir with the provided message.
// func Commit(dir string, message string) (string, string, error) {
// 	out, errOut, code, err := RunGitExitCode(dir, "commit", "-m", message)
// 	if err != nil {
// 		return out, errOut, err
// 	}
// 	if code != 0 {
// 		return out, errOut, fmt.Errorf("commit failed: exit %d", code)
// 	}
// 	return out, errOut, nil
// }

// // Pull performs a git pull in dir. go-git has limited support for pull; we provide
// // a best-effort implementation.
// func Pull(dir string, args ...string) (string, string, error) {
// 	repo, err := git.PlainOpen(dir)
// 	if err != nil {
// 		return "", err.Error(), err
// 	}
// 	wt, err := repo.Worktree()
// 	if err != nil {
// 		return "", err.Error(), err
// 	}
// 	// Default to origin and current branch
// 	head, herr := repo.Head()
// 	if herr != nil {
// 		return "", herr.Error(), herr
// 	}
// 	remoteName := "origin"
// 	ref := head.Name()
// 	err = wt.Pull(&git.PullOptions{RemoteName: remoteName, ReferenceName: ref})
// 	if err != nil && err != git.NoErrAlreadyUpToDate {
// 		return "", err.Error(), err
// 	}
// 	return "", "", nil
// }

// // Push performs a git push in dir. We implement a basic push to origin.
// func Push(dir string, args ...string) (string, string, error) {
// 	repo, err := git.PlainOpen(dir)
// 	if err != nil {
// 		return "", err.Error(), err
// 	}
// 	err = repo.Push(&git.PushOptions{})
// 	if err != nil && err != git.NoErrAlreadyUpToDate {
// 		return "", err.Error(), err
// 	}
// 	return "", "", nil
// }

// CurrentBranch returns the current branch short name.
func CurrentBranch(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}
	return head.Name().Short(), "", nil
}

// RemoteURL returns the URL for the default remote "origin".
func RemoteURL(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	remotes, rerr := repo.Remotes()
	if rerr != nil {
		return "", rerr.Error(), rerr
	}
	for _, rem := range remotes {
		if rem.Config().Name == "origin" {
			urls := rem.Config().URLs
			if len(urls) > 0 {
				return strings.TrimSpace(urls[0]), "", nil
			}
		}
	}
	return "", "", fmt.Errorf("origin not found")
}

// RevParse resolves a revision to a commit SHA.
func RevParse(dir, rev string) (string, string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", "", errors.New("rev must not be empty")
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	// Use ResolveRevision which accepts many ref syntaxes.
	r, rerr := repo.ResolveRevision(plumbing.Revision(rev))
	if rerr != nil {
		return "", rerr.Error(), rerr
	}
	return r.String(), "", nil
}

// ListBranches lists local branch names, one per line.
func ListBranches(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	iter, err := repo.Branches()
	if err != nil {
		return "", err.Error(), err
	}
	var buf bytes.Buffer
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		buf.WriteString(ref.Name().Short() + "\n")
		return nil
	})
	if err != nil {
		return "", err.Error(), err
	}
	return buf.String(), "", nil
}

// ListRemotes lists remotes in a format similar to `git remote -v`.
func ListRemotes(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	remotes, rerr := repo.Remotes()
	if rerr != nil {
		return "", rerr.Error(), rerr
	}
	var buf bytes.Buffer
	for _, r := range remotes {
		for _, u := range r.Config().URLs {
			buf.WriteString(fmt.Sprintf("%s\t%s (fetch)\n", r.Config().Name, u))
			buf.WriteString(fmt.Sprintf("%s\t%s (push)\n", r.Config().Name, u))
		}
	}
	return buf.String(), "", nil
}

// LatestCommit returns the latest commit as a single line: "<hash> <subject>".
func LatestCommit(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}
	commit, cerr := repo.CommitObject(head.Hash())
	if cerr != nil {
		return "", cerr.Error(), cerr
	}
	subject := strings.SplitN(commit.Message, "\n", 2)[0]
	return fmt.Sprintf("%s %s", commit.Hash.String(), strings.TrimSpace(subject)), "", nil
}

// ShowFileAtRev returns the file content at the given revision.
func ShowFileAtRev(dir, rev, path string) (string, string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", "", errors.New("rev must not be empty")
	}
	if strings.TrimSpace(path) == "" {
		return "", "", errors.New("path must not be empty")
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	r, rerr := repo.ResolveRevision(plumbing.Revision(rev))
	if rerr != nil {
		return "", rerr.Error(), rerr
	}
	commit, cerr := repo.CommitObject(*r)
	if cerr != nil {
		return "", cerr.Error(), cerr
	}
	tree, terr := commit.Tree()
	if terr != nil {
		return "", terr.Error(), terr
	}
	file, ferr := tree.File(path)
	if ferr != nil {
		return "", ferr.Error(), ferr
	}
	reader, rerr2 := file.Blob.Reader()
	if rerr2 != nil {
		return "", rerr2.Error(), rerr2
	}
	defer reader.Close()
	b, rerr3 := io.ReadAll(reader)
	if rerr3 != nil {
		return "", rerr3.Error(), rerr3
	}
	return string(b), "", nil
}
