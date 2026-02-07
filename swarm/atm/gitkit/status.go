package gitkit

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Status is the public entrypoint used by callers. It returns a verbose multi-line
// status as a string and preserves the previous signature (stdout, stderr, error).
func Status(dir string) (string, string, error) {
	out, err := statusDetailed(dir)
	if err != nil {
		return "", err.Error(), err
	}
	return out, "", nil
}

// statusDetailed implements a verbose git-like status for the repository at dir.
// It intentionally avoids relying on go-git's internal Status struct layout and
// returns a human readable multi-line string. The public, compatibility Status
// function delegates to this helper.
func statusDetailed(dir string) (string, error) {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return "", fmt.Errorf("failed to open repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	st, err := wt.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	var lines []string

	// Branch / HEAD line
	if head.Name().IsBranch() {
		lines = append(lines, fmt.Sprintf("On branch %s", head.Name().Short()))
	} else {
		shortHash := head.Hash().String()
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		lines = append(lines, fmt.Sprintf("HEAD detached at %s", shortHash))
	}

	// Upstream / ahead-behind
	if head.Name().IsBranch() {
		branchName := head.Name().Short()
		if cfg, err := repo.Config(); err == nil {
			if section, ok := cfg.Branches[branchName]; ok && section.Remote != "" && section.Merge != "" {
				remote := section.Remote
				mergeRef := plumbing.ReferenceName(section.Merge)
				mergeBranch := mergeRef.Short()
				upstreamRefName := plumbing.NewRemoteReferenceName(remote, mergeBranch)
				upRef, err := repo.Reference(upstreamRefName, true)
				if err == nil {
					localHash := head.Hash()
					upHash := upRef.Hash()
					if localHash != upHash {
						ahead, behind, divErr := countDivergence(repo, localHash, upHash)
						if divErr == nil {
							if ahead > 0 {
								pl := plural(ahead)
								lines = append(lines, fmt.Sprintf("Your branch is ahead of '%s' by %d commit%s.", upstreamRefName.Short(), ahead, pl))
								lines = append(lines, "[use \"git push\" to publish your local commits]")
							}
							if behind > 0 {
								pl := plural(behind)
								lines = append(lines, fmt.Sprintf("Your branch is behind '%s' by %d commit%s.", upstreamRefName.Short(), behind, pl))
								lines = append(lines, "[use \"git pull\" to update your local branch]")
							}
						}
					}
				}
			}
		}
	}

	stStr := strings.TrimSpace(st.String())
	if stStr == "" {
		lines = append(lines, "nothing to commit, working tree clean")
	} else {
		// Include the porcelain status block as-is under a header so tests can
		// still assert on human readable lines while keeping the detailed
		// porcelain output available.
		lines = append(lines, stStr)
	}

	return strings.Join(lines, "\n"), nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func reachableFrom(repo *git.Repository, start plumbing.Hash) (map[plumbing.Hash]struct{}, error) {
	seen := make(map[plumbing.Hash]struct{})
	stack := []plumbing.Hash{start}
	for len(stack) > 0 {
		h := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		c, err := repo.CommitObject(h)
		if err != nil {
			continue
		}
		for _, ph := range c.ParentHashes {
			stack = append(stack, ph)
		}
	}
	return seen, nil
}

func countDivergence(repo *git.Repository, local, upstream plumbing.Hash) (int, int, error) {
	upSeen, err := reachableFrom(repo, upstream)
	if err != nil {
		return 0, 0, err
	}
	iter, err := repo.Log(&git.LogOptions{From: local})
	if err != nil {
		return 0, 0, err
	}
	defer iter.Close()
	ahead := 0
	for {
		c, err := iter.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ahead, 0, err
		}
		if _, ok := upSeen[c.Hash]; !ok {
			ahead++
		}
	}
	localSeen, err := reachableFrom(repo, local)
	if err != nil {
		return ahead, 0, err
	}
	iter2, err := repo.Log(&git.LogOptions{From: upstream})
	if err != nil {
		return ahead, 0, err
	}
	defer iter2.Close()
	behind := 0
	for {
		c, err := iter2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ahead, behind, err
		}
		if _, ok := localSeen[c.Hash]; !ok {
			behind++
		}
	}
	return ahead, behind, nil
}
