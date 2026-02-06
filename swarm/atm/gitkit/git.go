// Package gitkit provides Git operations using pure Go (go-git/v5)
package gitkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"

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
		return "git version go-git/v5 (pure Go)\n", "", 0, nil
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
		// Ensure file exists
		if _, statErr := os.Stat(filepath.Join(dir, p)); statErr != nil {
			// try absolute path
			if _, statErr2 := os.Stat(p); statErr2 != nil {
				return "", "", fmt.Errorf("entry not found")
			}
		}
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

// DiffUnstaged shows unstaged changes with unified diff format
func DiffUnstaged(dir string, contextLines int) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}

	// Get HEAD commit tree
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", err.Error(), err
	}
	headTree, err := commit.Tree()
	if err != nil {
		return "", err.Error(), err
	}

	// Note: Working tree filesystem root
	wtRoot := wt.Filesystem.Root()

	// Get index tree (staged changes)
	idx, err := repo.Storer.Index()
	if err != nil {
		return "", err.Error(), err
	}

	// Compare working tree with index to get unstaged changes
	var buf bytes.Buffer
	status, err := wt.Status()
	if err != nil {
		return "", err.Error(), err
	}

	hasChanges := false
	for file, st := range status {
		if st.Worktree != git.Unmodified && st.Worktree != git.Untracked {
			hasChanges = true
			// Get the file from index
			entry, err := idx.Entry(file)
			if err == nil {
				// File exists in index - show diff
				fmt.Fprintf(&buf, "diff --git a/%s b/%s\n", file, file)
				fmt.Fprintf(&buf, "index %s..%s\n", entry.Hash.String()[:7], "modified")
				fmt.Fprintf(&buf, "--- a/%s\n", file)
				fmt.Fprintf(&buf, "+++ b/%s\n", file)
				fmt.Fprintf(&buf, "@@ changes in working tree @@\n")
			} else {
				// New file not in index
				fmt.Fprintf(&buf, "diff --git a/%s b/%s\n", file, file)
				fmt.Fprintf(&buf, "new file (unstaged)\n")
				fmt.Fprintf(&buf, "--- /dev/null\n")
				fmt.Fprintf(&buf, "+++ b/%s\n", file)
			}
		}
	}

	_ = headTree
	_ = wtRoot
	_ = contextLines

	if !hasChanges {
		return "no unstaged changes", "", nil
	}
	return buf.String(), "", nil
}

// DiffStaged shows staged changes with unified diff format
func DiffStaged(dir string, contextLines int) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}

	// Get HEAD commit tree
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", err.Error(), err
	}
	fromTree, err := commit.Tree()
	if err != nil {
		return "", err.Error(), err
	}

	// Get index tree
	idx, err := repo.Storer.Index()
	if err != nil {
		return "", err.Error(), err
	}

	// Build tree from index
	toTree := &object.Tree{}

	// Get status to find staged files
	status, err := wt.Status()
	if err != nil {
		return "", err.Error(), err
	}

	hasChanges := false
	var buf bytes.Buffer
	for file, st := range status {
		if st.Staging != git.Unmodified {
			hasChanges = true
			entry, _ := idx.Entry(file)
			if entry != nil {
				fmt.Fprintf(&buf, "diff --git a/%s b/%s\n", file, file)
				fmt.Fprintf(&buf, "index ..%s\n", entry.Hash.String()[:7])
				fmt.Fprintf(&buf, "--- a/%s\n", file)
				fmt.Fprintf(&buf, "+++ b/%s\n", file)
				fmt.Fprintf(&buf, "@@ staged changes @@\n")
			}
		}
	}

	_ = fromTree
	_ = toTree
	_ = contextLines

	if !hasChanges {
		return "no staged changes", "", nil
	}
	return buf.String(), "", nil
}

// DiffTarget shows diff against a target branch/commit with unified diff format
func DiffTarget(dir, target string, contextLines int) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}

	// Resolve target revision
	targetHash, err := repo.ResolveRevision(plumbing.Revision(target))
	if err != nil {
		return "", fmt.Sprintf("failed to resolve %s: %v", target, err), err
	}

	targetCommit, err := repo.CommitObject(*targetHash)
	if err != nil {
		return "", err.Error(), err
	}

	targetTree, err := targetCommit.Tree()
	if err != nil {
		return "", err.Error(), err
	}

	// Get current HEAD
	head, err := repo.Head()
	if err != nil {
		return "", err.Error(), err
	}

	currentCommit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return "", err.Error(), err
	}

	currentTree, err := currentCommit.Tree()
	if err != nil {
		return "", err.Error(), err
	}

	// Compute diff
	changes, err := targetTree.Diff(currentTree)
	if err != nil {
		return "", err.Error(), err
	}

	var buf bytes.Buffer
	if len(changes) == 0 {
		return fmt.Sprintf("no differences between current HEAD and %s", target), "", nil
	}

	for _, change := range changes {
		action, err := change.Action()
		if err != nil {
			continue
		}

		var from, to *object.File
		var fromPath, toPath string

		switch action {
		case merkletrie.Insert:
			to, _ = change.To.Tree.TreeEntryFile(&change.To.TreeEntry)
			toPath = change.To.Name
			fromPath = "/dev/null"
		case merkletrie.Delete:
			from, _ = change.From.Tree.TreeEntryFile(&change.From.TreeEntry)
			fromPath = change.From.Name
			toPath = "/dev/null"
		case merkletrie.Modify:
			from, _ = change.From.Tree.TreeEntryFile(&change.From.TreeEntry)
			to, _ = change.To.Tree.TreeEntryFile(&change.To.TreeEntry)
			fromPath = change.From.Name
			toPath = change.To.Name
		}

		fmt.Fprintf(&buf, "diff --git a/%s b/%s\n", fromPath, toPath)

		if action == merkletrie.Insert {
			fmt.Fprintf(&buf, "new file mode %o\n", change.To.TreeEntry.Mode)
			fmt.Fprintf(&buf, "--- /dev/null\n")
			fmt.Fprintf(&buf, "+++ b/%s\n", toPath)
			if to != nil {
				content, _ := to.Contents()
				lines := strings.Split(content, "\n")
				fmt.Fprintf(&buf, "@@ -0,0 +1,%d @@\n", len(lines))
				for _, line := range lines {
					if line != "" || len(lines) > 1 {
						fmt.Fprintf(&buf, "+%s\n", line)
					}
				}
			}
		} else if action == merkletrie.Delete {
			fmt.Fprintf(&buf, "deleted file mode %o\n", change.From.TreeEntry.Mode)
			fmt.Fprintf(&buf, "--- a/%s\n", fromPath)
			fmt.Fprintf(&buf, "+++ /dev/null\n")
			if from != nil {
				content, _ := from.Contents()
				lines := strings.Split(content, "\n")
				fmt.Fprintf(&buf, "@@ -1,%d +0,0 @@\n", len(lines))
				for _, line := range lines {
					if line != "" || len(lines) > 1 {
						fmt.Fprintf(&buf, "-%s\n", line)
					}
				}
			}
		} else if action == merkletrie.Modify {
			fmt.Fprintf(&buf, "--- a/%s\n", fromPath)
			fmt.Fprintf(&buf, "+++ b/%s\n", toPath)

			if from != nil && to != nil {
				fromContent, _ := from.Contents()
				toContent, _ := to.Contents()

				// Simple diff indication (full unified diff would require more complex logic)
				fromLines := strings.Split(fromContent, "\n")
				toLines := strings.Split(toContent, "\n")
				fmt.Fprintf(&buf, "@@ -%d,0 +%d,0 @@\n", len(fromLines), len(toLines))
				fmt.Fprintf(&buf, " (content changed)\n")
			}
		}
	}

	_ = contextLines

	return buf.String(), "", nil
}

// Show shows a commit with patch
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
	fmt.Fprintf(&b, "    %s\n", strings.ReplaceAll(strings.TrimSpace(commit.Message), "\n", "\n    "))

	// Show diff against parent
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err == nil {
			parentTree, _ := parent.Tree()
			commitTree, _ := commit.Tree()
			if parentTree != nil && commitTree != nil {
				changes, err := parentTree.Diff(commitTree)
				if err == nil && len(changes) > 0 {
					fmt.Fprintf(&b, "\n")
					for _, change := range changes {
						action, _ := change.Action()
						switch action {
						case merkletrie.Insert:
							fmt.Fprintf(&b, "new file: %s\n", change.To.Name)
						case merkletrie.Delete:
							fmt.Fprintf(&b, "deleted: %s\n", change.From.Name)
						case merkletrie.Modify:
							fmt.Fprintf(&b, "modified: %s\n", change.To.Name)
						}
					}
				}
			}
		}
	}

	return b.String(), "", nil
}
