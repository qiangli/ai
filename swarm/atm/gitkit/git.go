// Package gitkit provides Git operations using pure Go (go-git/v5)
package gitkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/qiangli/ai/swarm/api"
)

// ExecGit is a wrapper that returns stdout, stderr, and error
func ExecGit(dir string, args ...string) (stdout string, stderr string, err error) {
	out, errOut, _, runErr := RunGitExitCode(dir, args...)
	if runErr != nil {
		return out, errOut, fmt.Errorf("git %v failed: %w", args, runErr)
	}
	return out, errOut, nil
}

// RunGitExitCode executes git commands and returns stdout, stderr, exit code, and error
func RunGitExitCode(dir string, args ...string) (stdout string, stderr string, exitCode int, err error) {
	if len(args) == 0 {
		return "", "", 2, fmt.Errorf("no git command provided")
	}
	cmd := args[0]
	switch cmd {
	case "--version":
		return "git version go-git (pure Go)\n", "", 0, nil
	case "status":
		out, errOut, err := Status(dir)
		if err != nil {
			return out, errOut, 1, err
		}
		return out, errOut, 0, nil
	case "add":
		if len(args) < 2 {
			return "", "", 2, fmt.Errorf("add requires file arguments")
		}
		out, errOut, err := Add(dir, args[1:])
		if err != nil {
			return out, errOut, 1, err
		}
		return out, errOut, 0, nil
	case "commit":
		msg := ""
		cmdArgs := []string{}
		for i := 1; i < len(args); i++ {
			if args[i] == "-m" && i+1 < len(args) {
				msg = args[i+1]
				i++
			} else {
				cmdArgs = append(cmdArgs, args[i])
			}
		}
		out, errOut, code, err := Commit(dir, msg, cmdArgs)
		return out, errOut, code, err
	default:
		return "", "", 127, fmt.Errorf("unsupported git subcommand: %s", cmd)
	}
}

// Status returns the working tree status
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
	return st.String(), "", nil
}

// Add stages files for commit
func Add(dir string, files []string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	for _, p := range files {
		if _, aerr := wt.Add(p); aerr != nil {
			return "", aerr.Error(), aerr
		}
	}
	return fmt.Sprintf("added %d file(s)", len(files)), "", nil
}

// Commit creates a new commit with staged changes
func Commit(dir string, message string, args []string) (string, string, int, error) {
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

	var msg = message
	for i := 0; i < len(filtered); i++ {
		if filtered[i] == "-m" && i+1 < len(filtered) {
			msg = api.Cat(message, filtered[i+1], "\n")
			break
		}
	}
	if strings.TrimSpace(msg) == "" {
		return "", "", 2, fmt.Errorf("commit message must not be empty")
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), 1, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), 1, err
	}
	if name == "" || email == "" {
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
			name = "auto"
		}
		if email == "" {
			email = "gitkit@dhnt.io"
		}
	}
	commitHash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{Name: name, Email: email, When: time.Now()},
	})
	if err != nil {
		return "", err.Error(), 1, err
	}
	return commitHash.String(), "", 0, nil
}

// Pull fetches and merges changes from remote
func Pull(dir string, _ []string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}
	remoteName := "origin"
	ref := head.Name()
	err = wt.Pull(&git.PullOptions{RemoteName: remoteName, ReferenceName: ref})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err.Error(), err
	}
	return "pulled successfully", "", nil
}

// Push pushes changes to remote
func Push(dir string, _ []string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	err = repo.Push(&git.PushOptions{})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err.Error(), err
	}
	return "pushed successfully", "", nil
}

// CurrentBranch returns the name of the current branch
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

// RemoteURL returns the URL of the origin remote
func RemoteURL(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return "", err.Error(), err
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

// RevParse resolves a revision to a commit hash
func RevParse(dir, rev string) (string, string, error) {
	if strings.TrimSpace(rev) == "" {
		return "", "", errors.New("rev must not be empty")
	}
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	r, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return "", err.Error(), err
	}
	return r.String(), "", nil
}

// ListBranches lists all local branches
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

// ListRemotes lists all remotes
func ListRemotes(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return "", err.Error(), err
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

// LatestCommit returns the hash and subject of the most recent commit
func LatestCommit(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", err.Error(), err
	}
	subject := strings.SplitN(commit.Message, "\n", 2)[0]
	return fmt.Sprintf("%s %s", commit.Hash.String(), strings.TrimSpace(subject)), "", nil
}

// ShowFileAtRev shows the content of a file at a specific revision
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
	r, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return "", err.Error(), err
	}
	commit, err := repo.CommitObject(*r)
	if err != nil {
		return "", err.Error(), err
	}
	tree, err := commit.Tree()
	if err != nil {
		return "", err.Error(), err
	}
	file, err := tree.File(path)
	if err != nil {
		return "", err.Error(), err
	}
	reader, err := file.Blob.Reader()
	if err != nil {
		return "", err.Error(), err
	}
	defer reader.Close()
	b, err := io.ReadAll(reader)
	if err != nil {
		return "", err.Error(), err
	}
	return string(b), "", nil
}

// Clone clones a repository
func Clone(repoURL, destDir string) error {
	_, err := git.PlainClone(destDir, false, &git.CloneOptions{URL: repoURL})
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}
	return nil
}

// CommitLog represents a single commit entry
type CommitLog struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

// ParseTime parses various time formats
func ParseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t
	}
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
	return time.Time{}
}

// Log returns commit history
func Log(dir string, maxCount int, startT, endT time.Time) ([]CommitLog, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err.Error(), err
	}
	ref, err := repo.Head()
	if err != nil {
		return nil, err.Error(), err
	}
	logOpts := &git.LogOptions{From: ref.Hash()}
	iter, err := repo.Log(logOpts)
	if err != nil {
		return nil, err.Error(), err
	}
	max := 10
	if maxCount > 0 {
		max = maxCount
	}
	var logs []CommitLog
	err = iter.ForEach(func(c *object.Commit) error {
		if !startT.IsZero() && c.Author.When.Before(startT) {
			return nil
		}
		if !endT.IsZero() && c.Author.When.After(endT) {
			return nil
		}
		logs = append(logs, CommitLog{
			Hash:    c.Hash.String(),
			Author:  c.Author.Name,
			Date:    c.Author.When.Format(time.RFC3339),
			Message: c.Message,
		})
		if len(logs) >= max {
			return errors.New("stop")
		}
		return nil
	})
	if err != nil && err.Error() != "stop" {
		return nil, "", err
	}
	return logs, "", nil
}

// Reset unstages all changes
func Reset(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	err = wt.Reset(&git.ResetOptions{Mode: git.MixedReset})
	return "unstaged all changes", "", err
}

// Checkout switches to a different branch
func Checkout(dir, branchName string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	ref := plumbing.NewBranchReferenceName(branchName)
	err = wt.Checkout(&git.CheckoutOptions{
		Branch: ref,
	})
	return fmt.Sprintf("checked out branch %s", branchName), "", err
}

// CreateBranch creates a new branch
func CreateBranch(dir, branchName, baseBranch string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	
	var baseHash plumbing.Hash
	if baseBranch == "" {
		ref, err := repo.Head()
		if err != nil {
			return "", err.Error(), err
		}
		baseHash = ref.Hash()
	} else {
		h, err := repo.ResolveRevision(plumbing.Revision(baseBranch))
		if err != nil {
			return "", err.Error(), err
		}
		baseHash = *h
	}
	
	// Create branch reference
	refName := plumbing.NewBranchReferenceName(branchName)
	ref := plumbing.NewHashReference(refName, baseHash)
	err = repo.Storer.SetReference(ref)
	if err != nil {
		return "", err.Error(), err
	}
	
	return fmt.Sprintf("created branch %s", branchName), "", nil
}

// Branches lists branches
func Branches(dir, branchType string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	var ls []string
	switch branchType {
	case "local", "":
		iter, err := repo.Branches()
		if err != nil {
			return "", err.Error(), err
		}
		iter.ForEach(func(ref *plumbing.Reference) error {
			ls = append(ls, ref.Name().Short())
			return nil
		})
	case "remote":
		refs, err := repo.References()
		if err != nil {
			return "", err.Error(), err
		}
		refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Name().IsRemote() {
				ls = append(ls, ref.Name().Short())
			}
			return nil
		})
	case "all":
		iter, _ := repo.Branches()
		iter.ForEach(func(ref *plumbing.Reference) error {
			ls = append(ls, ref.Name().Short())
			return nil
		})
		refs, _ := repo.References()
		refs.ForEach(func(ref *plumbing.Reference) error {
			if ref.Name().IsRemote() {
				ls = append(ls, ref.Name().Short())
			}
			return nil
		})
	}
	bs, _ := json.Marshal(ls)
	return string(bs), "", nil
}

// DiffUnstaged shows unstaged changes
func DiffUnstaged(dir string, _ int) (string, string, error) {
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
	var b strings.Builder
	for file, status := range st {
		if status.Worktree != git.Unmodified && status.Worktree != git.Untracked {
			fmt.Fprintf(&b, "%c %s\n", status.Worktree, file)
		}
	}
	if b.Len() == 0 {
		return "no unstaged changes", "", nil
	}
	return b.String(), "", nil
}

// DiffStaged shows staged changes
func DiffStaged(dir string, _ int) (string, string, error) {
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
	var b strings.Builder
	for file, status := range st {
		if status.Staging != git.Unmodified {
			fmt.Fprintf(&b, "%c %s\n", status.Staging, file)
		}
	}
	if b.Len() == 0 {
		return "no staged changes", "", nil
	}
	return b.String(), "", nil
}

// DiffTarget shows diff against a target branch/commit
func DiffTarget(dir, target string, _ int) (string, string, error) {
	h, errStr, err := RevParse(dir, target)
	if err != nil {
		return "", errStr, err
	}
	return fmt.Sprintf("diff vs %s:\ncommit: %s", target, h), "", nil
}

// Show shows a commit
func Show(dir, rev string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	h, err := repo.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return "", err.Error(), err
	}
	commit, err := repo.CommitObject(*h)
	if err != nil {
		return "", err.Error(), err
	}
	
	var b strings.Builder
	fmt.Fprintf(&b, "commit %s\n", commit.Hash.String())
	fmt.Fprintf(&b, "Author: %s <%s>\n", commit.Author.Name, commit.Author.Email)
	fmt.Fprintf(&b, "Date:   %s\n\n", commit.Author.When.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "    %s\n", strings.ReplaceAll(commit.Message, "\n", "\n    "))
	
	return b.String(), "", nil
}
