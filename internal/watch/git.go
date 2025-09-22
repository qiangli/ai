package watch

import (
	"context"
	"os"
	"path/filepath"
	"time"

	git "github.com/go-git/go-git/v5"

	"github.com/qiangli/ai/agent"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// TODO custom prefix for different file types
var prefixMap = map[string]string{
	".go":  "//",
	".py":  "#",
	".ts":  "//",
	".tsx": "//",
	".sh":  "#",
	".md":  ">",
}

func WatchRepo(ctx context.Context, cfg *api.AppConfig) error {
	repoPath := filepath.Clean(cfg.Workspace)

	log.GetLogger(ctx).Debug("Watching git repository: %s\n", repoPath)

	if err := os.Chdir(repoPath); err != nil {
		return err
	}

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	sizeMemo := make(map[string]int64)

	run := func(path string) {
		ext := filepath.Ext(path)
		prefix, ok := prefixMap[ext]
		if !ok {
			log.GetLogger(ctx).Debug("Unsupported file extension: %s, use default prefix\n", ext)
			prefix = "#"
		}
		line, err := parseFile(path, prefix)
		if err != nil {
			log.GetLogger(ctx).Error("Error parsing file: %s\n", err)
			return
		}

		if len(line) == 0 {
			log.GetLogger(ctx).Debug("No command found in file: %s\n", path)
			return
		}

		in, err := parseUserInput(line, prefix)
		if err != nil {
			log.GetLogger(ctx).Error("Error parsing user input: %s\n", err)
			return
		}
		if in.Agent == "" {
			in.Agent = cfg.Agent
		}
		// if in.Command == "" {
		// 	in.Command = cfg.Command
		// }

		log.GetLogger(ctx).Debug("agent: %s\n", in.Agent)

		cfg.Format = "text"
		if err := agent.RunSwarm(ctx, cfg, in); err != nil {
			log.GetLogger(ctx).Error("Error running agent: %s\n", err)
			return
		}

		//success
		log.GetLogger(ctx).Info("ai executed successfully: %s\n", line)
		if err := replaceContentInFile(path, line, prefix, cfg.Stdout); err != nil {
			log.GetLogger(ctx).Error("Error replacing content in file: %s\n", err)
			return
		}
	}

	// Unmodified         StatusCode = ' '
	// Untracked          StatusCode = '?'
	// Modified           StatusCode = 'M'
	// Added              StatusCode = 'A'
	// Deleted            StatusCode = 'D'
	// Renamed            StatusCode = 'R'
	// Copied             StatusCode = 'C'
	// UpdatedButUnmerged StatusCode = 'U'
	check := func() {
		st, err := worktree.Status()
		if err != nil {
			log.GetLogger(ctx).Error("Error getting worktree status: %s\n", err)
			return
		}

		for path, s := range st {
			if s.Worktree == git.Modified {
				sz, ok := sizeMemo[path]
				if !ok {
					log.GetLogger(ctx).Debug("%c %c  %s\n", rune(s.Staging), rune(s.Worktree), path)
				}

				abs := filepath.Join(repoPath, path)
				info, err := os.Stat(abs)
				if err != nil {
					log.GetLogger(ctx).Error("Error getting file info: %s\n", err)
					continue
				}
				nz := info.Size()
				if nz != sz {
					sizeMemo[path] = info.Size()
					log.GetLogger(ctx).Info("***  %s\n", path)
					run(path)
				}
			}
		}
	}

	interval := 1800 * time.Millisecond

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		<-ticker.C
		check()
	}
}
