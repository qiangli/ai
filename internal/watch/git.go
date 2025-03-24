package watch

import (
	"fmt"

	git "github.com/go-git/go-git/v5"

	"github.com/qiangli/ai/internal/log"
)

func WatchGit(repoPath string) error {
	// Open the repository
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	// Get the worktree
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	// Get the status of the worktree
	st, err := worktree.Status()
	if err != nil {
		return err
	}

	// Iterate through the status to find modified/changed files
	// fmt.Println("Changed files:")
	// Unmodified         StatusCode = ' '
	// Untracked          StatusCode = '?'
	// Modified           StatusCode = 'M'
	// Added              StatusCode = 'A'
	// Deleted            StatusCode = 'D'
	// Renamed            StatusCode = 'R'
	// Copied             StatusCode = 'C'
	// UpdatedButUnmerged StatusCode = 'U'
	for path, s := range st {
		if s.Staging != git.Added && s.Staging != git.Renamed || s.Worktree != git.Unmodified {
			fmt.Printf("- %s: %+v\n", path, s.Worktree)
		}
	}

	log.Infof("Watching git repository at %s %s", repoPath, worktree.Filesystem.Root)
	return nil
}
