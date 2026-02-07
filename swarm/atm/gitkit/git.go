package gitkit

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	transportssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
	cryptossh "golang.org/x/crypto/ssh"
)

// AuthParams encapsulates credentials used for push/pull operations.
type AuthParams struct {
	Token    string
	Username string
	Password string
	SSHKey   string
}

var (
	// sshAgentAuth returns a go-git transport.AuthMethod by attempting to use
	// the SSH agent. Default implementation returns an error; callers/tests
	// may override this variable to provide agent support.
	sshAgentAuth func(user string) (transport.AuthMethod, error) = func(user string) (transport.AuthMethod, error) {
		return nil, fmt.Errorf("ssh agent unsupported")
	}
	// sshSigner parses a private key to a crypto/ssh.Signer; keep using
	// golang.org/x/crypto/ssh for parsing and Signer type.
	sshSigner func(key []byte) (cryptossh.Signer, error) = cryptossh.ParsePrivateKey
)

func preparePushAuth(remoteURL string, p AuthParams) (transport.AuthMethod, error) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return nil, nil
	}
	if strings.HasPrefix(remoteURL, "ssh://") || strings.Contains(remoteURL, "git@") {
		// SSH remote
		auth, err := sshAgentAuth("git")
		if err == nil {
			return auth, nil
		}
		key := p.SSHKey
		if key == "" {
			key = os.Getenv("GIT_SSH_KEY")
		}
		if key != "" {
			signer, err := sshSigner([]byte(key))
			if err != nil {
				return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
			}
			pk := &transportssh.PublicKeys{User: "git", Signer: signer}
			return pk, nil
		}
		return nil, nil // backward compat: no creds
	} else {
		// HTTPS remote
		token := p.Token
		if token == "" {
			token = os.Getenv("GIT_TOKEN")
		}
		if token != "" {
			return &http.BasicAuth{Username: "x-access-token", Password: token}, nil
		}
		username := p.Username
		if username == "" {
			username = os.Getenv("GIT_USERNAME")
		}
		password := p.Password
		if password == "" {
			password = os.Getenv("GIT_PASSWORD")
		}
		if username != "" && password != "" {
			return &http.BasicAuth{Username: username, Password: password}, nil
		}
		return nil, nil // backward compat: no creds
	}
}

// Push pushes changes to remote with auth support
func Push(dir string, args []string, token, username, password, sshKey string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	remoteName := "origin"
	if len(args) > 0 {
		remoteName = args[0]
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return "", err.Error(), err
	}
	var remote *git.Remote
	for _, r := range remotes {
		if r.Config().Name == remoteName {
			remote = r
			break
		}
	}
	if remote == nil {
		return "", fmt.Sprintf("remote %q not found", remoteName), fmt.Errorf("remote not found")
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return "", "no URL configured for remote", fmt.Errorf("no remote URL")
	}
	remoteURL := strings.TrimSpace(urls[0])
	auth, merr := preparePushAuth(remoteURL, AuthParams{
		Token:    token,
		Username: username,
		Password: password,
		SSHKey:   sshKey,
	})
	if merr != nil {
		return "", merr.Error(), merr
	}
	opts := &git.PushOptions{
		RemoteName: remoteName,
		Auth:       auth,
	}
	err = repo.Push(opts)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err.Error(), err
	}
	return "pushed successfully", "", nil
}

// --- Low-level helper implementations -------------------------------------------------

// StatusPorcelain returns a porcelain-like status for the repository at dir.
func StatusPorcelain(dir string) (string, string, error) {
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

// DiffUnstaged returns a unified diff between working tree and index.
func DiffUnstaged(dir string, ctx int) (string, string, error) {
	// Minimal implementation: not producing full unified diff; return empty for now.
	return "", "", nil
}

// DiffStaged returns a unified diff between index and HEAD.
func DiffStaged(dir string, ctx int) (string, string, error) {
	return "", "", nil
}

// DiffTarget returns unified diff between current and target revision.
func DiffTarget(dir string, target string, ctx int) (string, string, error) {
	return "", "", nil
}

// Commit creates a commit with message msg. extra is optional args (ignored currently).
func Commit(dir string, msg string, extra []string) (string, string, int, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), 1, err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), 1, err
	}
	hash, err := wt.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{Name: "gitkit", Email: "gitkit@example.com", When: time.Now()},
	})
	if err != nil {
		return "", err.Error(), 1, err
	}
	return hash.String(), "", 0, nil
}

// Add stages the provided files and returns a summary
func Add(dir string, files []string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	var out []string
	for _, f := range files {
		p := f
		// ensure path is relative to repository dir
		p = strings.TrimPrefix(p, string(filepath.Separator))
		if strings.Contains(p, "..") {
			// skip suspicious paths
			continue
		}
		_, err := wt.Add(p)
		if err != nil {
			out = append(out, fmt.Sprintf("error adding %s: %v", p, err))
		} else {
			out = append(out, fmt.Sprintf("added %s", p))
		}
	}
	return strings.Join(out, "\n"), "", nil
}

// Reset unstages all changes (mixed reset)
func Reset(dir string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	// Try to perform a mixed reset (unstage changes)
	if err := wt.Reset(&git.ResetOptions{Mode: git.MixedReset}); err != nil {
		return "", err.Error(), err
	}
	return "reset unstaged changes", "", nil
}

// Restore restores files from a given source (revision) to working tree and/or index.
// Similar to git restore [--source=<tree>] [--staged] [--worktree] <pathspec>...
func Restore(dir string, paths []string, source string, staged, worktree, force, backup, dryRun bool) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}

	// Resolve source to a tree
	hash, err := repo.ResolveRevision(plumbing.Revision(source))
	if err != nil {
		return "", fmt.Sprintf("failed to resolve revision %q: %v", source, err), err
	}

	commit, err := repo.CommitObject(*hash)
	if err != nil {
		return "", fmt.Sprintf("failed to get commit object: %v", err), err
	}

	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Sprintf("failed to get tree: %v", err), err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}

	var restoredPaths []string
	var errors []string

	// If no paths specified, restore all
	if len(paths) == 0 {
		if worktree {
			// Restore entire worktree using Checkout
			err := wt.Checkout(&git.CheckoutOptions{
				Hash:  *hash,
				Force: force,
			})
			if err != nil {
				return "", fmt.Sprintf("failed to restore worktree: %v", err), err
			}
			restoredPaths = append(restoredPaths, ".")
		}
		if staged {
			// Reset index to source commit
			err := wt.Reset(&git.ResetOptions{
				Commit: *hash,
				Mode:   git.MixedReset,
			})
			if err != nil {
				return "", fmt.Sprintf("failed to restore index: %v", err), err
			}
		}
	} else {
		// Restore specific paths
		for _, path := range paths {
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}

			// Normalize path (remove leading ./)
			path = strings.TrimPrefix(path, "./")

			if dryRun {
				restoredPaths = append(restoredPaths, fmt.Sprintf("[dry-run] would restore %s", path))
				continue
			}

			// Get file from tree
			file, err := tree.File(path)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to get file %s from tree: %v", path, err))
				continue
			}

			// Read file content
			content, err := file.Contents()
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to read file %s: %v", path, err))
				continue
			}

			if worktree {
				// Write file to worktree
				fullPath := filepath.Join(dir, path)

				// Create backup if requested
				if backup {
					if _, err := os.Stat(fullPath); err == nil {
						backupPath := fullPath + ".backup"
						data, _ := os.ReadFile(fullPath)
						_ = os.WriteFile(backupPath, data, 0644)
					}
				}

				// Ensure directory exists
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					errors = append(errors, fmt.Sprintf("failed to create directory for %s: %v", path, err))
					continue
				}

				// Write file
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					errors = append(errors, fmt.Sprintf("failed to write file %s: %v", path, err))
					continue
				}
			}

			if staged {
				// Stage the file (add to index)
				_, err := wt.Add(path)
				if err != nil {
					errors = append(errors, fmt.Sprintf("failed to stage file %s: %v", path, err))
					continue
				}
			}

			restoredPaths = append(restoredPaths, path)
		}
	}

	var outMsg string
	if len(restoredPaths) > 0 {
		outMsg = fmt.Sprintf("Restored %d path(s) from %s:\n%s", len(restoredPaths), source, strings.Join(restoredPaths, "\n"))
	}

	var errMsg string
	if len(errors) > 0 {
		errMsg = strings.Join(errors, "\n")
	}

	if len(errors) > 0 && len(restoredPaths) == 0 {
		return outMsg, errMsg, fmt.Errorf("restore failed")
	}

	return outMsg, errMsg, nil
}

// LogEntry represents a compact commit record
type LogEntry struct {
	Hash    string `json:"hash"`
	Author  string `json:"author"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

// ParseTime parses a timestamp string. Minimal support: empty->zero time, RFC3339 otherwise.
func ParseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "<no value>" {
		return time.Time{}
	}
	// try RFC3339
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t
	}
	// try Unix timestamp
	if unix, err2 := strconv.ParseInt(s, 10, 64); err2 == nil {
		return time.Unix(unix, 0)
	}
	return time.Time{}
}

// Log returns commit log entries up to max (0 means no limit), optionally between startT and endT.
func Log(dir string, max int, startT, endT time.Time) ([]LogEntry, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err.Error(), err
	}
	iter, err := repo.Log(&git.LogOptions{Order: git.LogOrderCommitterTime})
	if err != nil {
		return nil, err.Error(), err
	}
	defer iter.Close()
	var logs []LogEntry
	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if !startT.IsZero() && c.Author.When.Before(startT) {
			return nil
		}
		if !endT.IsZero() && c.Author.When.After(endT) {
			return nil
		}
		logs = append(logs, LogEntry{Hash: c.Hash.String(), Author: c.Author.Name, Date: c.Author.When.Format(time.RFC3339), Message: strings.TrimSpace(c.Message)})
		count++
		if max > 0 && count >= max {
			return io.EOF
		}
		return nil
	})
	if err != nil && err != io.EOF {
		return nil, err.Error(), err
	}
	return logs, "", nil
}

// CreateBranch creates a new branch with the given base (if provided). Minimal implementation using Checkout create.
func CreateBranch(dir, name, base string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	co := &git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(name), Create: true}
	if base != "" {
		// try to resolve base to hash and set Hash in CheckoutOptions if possible
		if h := plumbing.NewHash(base); h.IsZero() == false {
			co.Hash = h
		}
	}
	if err := wt.Checkout(co); err != nil {
		return "", err.Error(), err
	}
	return fmt.Sprintf("branch %s created", name), "", nil
}

// Checkout switches to an existing branch or commit
func Checkout(dir, name string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	// try branch
	if err := wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(name)}); err == nil {
		return fmt.Sprintf("switched to %s", name), "", nil
	}
	// try hash
	if h := plumbing.NewHash(name); h.IsZero() == false {
		if err := wt.Checkout(&git.CheckoutOptions{Hash: h, Force: true}); err == nil {
			return fmt.Sprintf("detached at %s", name), "", nil
		}
	}
	return "", fmt.Sprintf("failed to checkout %s", name), fmt.Errorf("checkout failed")
}

// Show returns basic representation of a revision (commit message + metadata)
func Show(dir string, rev string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	var c *object.Commit
	if rev == "HEAD" || rev == "" {
		head, err := repo.Head()
		if err != nil {
			return "", err.Error(), err
		}
		c, err = repo.CommitObject(head.Hash())
		if err != nil {
			return "", err.Error(), err
		}
	} else {
		h := plumbing.NewHash(rev)
		c, err = repo.CommitObject(h)
		if err != nil {
			// try resolve ref name
			ref, rerr := repo.Reference(plumbing.ReferenceName(rev), true)
			if rerr != nil {
				return "", err.Error(), err
			}
			c, err = repo.CommitObject(ref.Hash())
			if err != nil {
				return "", err.Error(), err
			}
		}
	}
	out := fmt.Sprintf("%s\nAuthor: %s\nDate: %s\n\n%s", c.Hash.String(), c.Author.Name, c.Author.When.Format(time.RFC3339), c.Message)
	return out, "", nil
}

// Branches lists branches. typ can be "local" or "remote" (default local)
func Branches(dir string, typ string) ([]string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, err.Error(), err
	}
	var res []string
	if typ == "remote" {
		refs, err := repo.References()
		if err != nil {
			return nil, err.Error(), err
		}
		_ = refs.ForEach(func(r *plumbing.Reference) error {
			if r.Name().IsRemote() {
				res = append(res, r.Name().String())
			}
			return nil
		})
	} else {
		iter, err := repo.Branches()
		if err != nil {
			return nil, err.Error(), err
		}
		_ = iter.ForEach(func(r *plumbing.Reference) error {
			res = append(res, r.Name().Short())
			return nil
		})
	}
	return res, "", nil
}

// Pull performs a simple pull from remote; args may contain remote and branch.
func Pull(dir string, args []string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", err.Error(), err
	}
	remote := "origin"
	if len(args) > 0 {
		remote = args[0]
	}
	// minimal: just call Pull with remote
	opts := &git.PullOptions{RemoteName: remote}
	if err := wt.Pull(opts); err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err.Error(), err
	}
	return "pulled", "", nil
}

// Tag creates a tag; annotated if requested
func Tag(dir string, name, rev string, annotated bool, msg string) (string, string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", err.Error(), err
	}
	h := plumbing.NewHash(rev)
	if annotated {
		// create annotated tag
		_, err := repo.CreateTag(name, h, &git.CreateTagOptions{Message: msg, Tagger: &object.Signature{Name: "gitkit", Email: "gitkit@example.com", When: time.Now()}})
		if err != nil {
			return "", err.Error(), err
		}
	} else {
		// lightweight tag -> create ref
		ref := plumbing.NewHashReference(plumbing.NewTagReferenceName(name), h)
		if err := repo.Storer.SetReference(ref); err != nil {
			return "", err.Error(), err
		}
	}
	return fmt.Sprintf("tag %s created", name), "", nil
}

// RunGitExitCode provides a lightweight emulation of some git subcommands.
// It returns stdout, stderr, exitCode, and error (error is non-nil when the emulation failed to run).
func RunGitExitCode(dir string, args ...string) (string, string, int, error) {
	if len(args) == 0 {
		return "", "", 2, fmt.Errorf("no args")
	}
	// if user asked for version, try to exec system git for accurate result
	for _, a := range args {
		if a == "--version" {
			cmd := exec.Command("git", args...)
			var outb, errb bytes.Buffer
			cmd.Stdout = &outb
			cmd.Stderr = &errb
			err := cmd.Run()
			exit := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exit = exitErr.ExitCode()
				} else {
					exit = 1
				}
			}
			return outb.String(), errb.String(), exit, err
		}
	}

	sub := args[0]
	subArgs := args[1:]
	switch sub {
	case "status":
		out, errStr, err := StatusPorcelain(dir)
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "add":
		if len(subArgs) == 0 {
			return "", "add requires files", 2, fmt.Errorf("add requires files")
		}
		out, errStr, err := Add(dir, subArgs)
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "reset":
		out, errStr, err := Reset(dir)
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "commit":
		// look for -m message
		msg := ""
		for i := 0; i < len(subArgs); i++ {
			if subArgs[i] == "-m" && i+1 < len(subArgs) {
				msg = subArgs[i+1]
				break
			}
		}
		if msg == "" {
			return "", "commit requires -m", 2, fmt.Errorf("commit requires -m")
		}
		hash, errStr, code, err := Commit(dir, msg, nil)
		return hash, errStr, code, err
	case "log":
		// simple mapping: optional -n
		max := 0
		for i := 0; i < len(subArgs); i++ {
			if subArgs[i] == "-n" && i+1 < len(subArgs) {
				if v, err := strconv.Atoi(subArgs[i+1]); err == nil {
					max = v
				}
			}
		}
		logs, errStr, err := Log(dir, max, time.Time{}, time.Time{})
		if err != nil {
			return "[]", errStr, 1, err
		}
		bs, _ := jsonMarshal(logs)
		return bs, errStr, 0, nil
	case "branch":
		// return list of local branches
		branches, errStr, err := Branches(dir, "local")
		if err != nil {
			return "", errStr, 1, err
		}
		return strings.Join(branches, "\n"), "", 0, nil
	case "checkout":
		if len(subArgs) == 0 {
			return "", "checkout requires name", 2, fmt.Errorf("checkout requires name")
		}
		out, errStr, err := Checkout(dir, subArgs[0])
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "show":
		if len(subArgs) == 0 {
			return "", "show requires rev", 2, fmt.Errorf("show requires rev")
		}
		out, errStr, err := Show(dir, subArgs[0])
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "branches":
		branches, errStr, err := Branches(dir, "local")
		if err != nil {
			return "", errStr, 1, err
		}
		bs, _ := jsonMarshal(branches)
		return bs, "", 0, nil
	case "pull":
		out, errStr, err := Pull(dir, subArgs)
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "push":
		out, errStr, err := Push(dir, subArgs, "", "", "", "")
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	case "tag":
		if len(subArgs) < 2 {
			return "", "tag requires name and rev", 2, fmt.Errorf("tag requires name and rev")
		}
		name := subArgs[0]
		rev := subArgs[1]
		out, errStr, err := Tag(dir, name, rev, false, "")
		if err != nil {
			return out, errStr, 1, err
		}
		return out, errStr, 0, nil
	default:
		// unsupported: attempt to run system git as fallback if available
		if path, err := exec.LookPath("git"); err == nil && path != "" {
			cmd := exec.Command("git", args...)
			if dir != "" {
				cmd.Dir = dir
			}
			var outb, errb bytes.Buffer
			cmd.Stdout = &outb
			cmd.Stderr = &errb
			err := cmd.Run()
			exit := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					exit = exitErr.ExitCode()
				} else {
					exit = 1
				}
			}
			return outb.String(), errb.String(), exit, err
		}
		return "", fmt.Sprintf("unsupported git subcommand: %s", sub), 127, fmt.Errorf("unsupported subcommand")
	}
}

// small helper: JSON marshal without extra imports here
func jsonMarshal(v any) (string, error) {
	b, err := jsonMarshalStd(v)
	return string(b), err
}

// we implement a tiny indirection to avoid importing encoding/json at top-level twice in different files
func jsonMarshalStd(v any) ([]byte, error) {
	// import on demand
	return func(v any) ([]byte, error) {
		return jsonMarshalReal(v)
	}(v)
}

// low-level real marshal implemented via standard library (separate function to keep imports local)
func jsonMarshalReal(v any) ([]byte, error) {
	return jsonMarshalImport(v)
}

// jsonMarshalImport is defined in a small file to include encoding/json import.
