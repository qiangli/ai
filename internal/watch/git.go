package watch

import (
	"os"
	"path/filepath"
	"time"

	git "github.com/go-git/go-git/v5"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/log"
)

func WatchRepo(cfg *internal.AppConfig) error {
	repoPath := filepath.Clean(cfg.Repo)

	log.Debugf("Watching git repository: %s\n", repoPath)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	sizeMemo := make(map[string]int64)

	// TODO custom prefix for different file types
	prefixMap := map[string]string{
		".go": "//",
		".py": "#",
		".sh": "#",
		".md": ">",
	}

	run := func(path string) {
		ext := filepath.Ext(path)
		prefix, ok := prefixMap[ext]
		if !ok {
			log.Debugf("Unsupported file extension: %s, use default prefix\n", ext)
			prefix = "#"
		}
		line, err := parseFile(path, prefix)
		if err != nil {
			log.Errorf("Error parsing file: %s\n", err)
			return
		}

		if len(line) == 0 {
			log.Debugf("No command found in file: %s\n", path)
			return
		}

		in, err := parseUserInput(line, prefix)
		if err != nil {
			log.Errorf("Error parsing user input: %s\n", err)
			return
		}
		if in.Agent == "" {
			in.Agent = cfg.Agent
		}
		if in.Command == "" {
			in.Command = cfg.Command
		}

		log.Debugf("agent: %s %s %v\n", in.Agent, in.Command)

		cfg.Format = "text"
		if err := agent.RunSwarm(cfg, in); err != nil {
			log.Errorf("Error running agent: %s\n", err)
			return
		}

		//success
		log.Infof("ai executed successfully: %s\n", line)
		if err := replaceContentInFile(path, line, prefix, cfg.Stdout); err != nil {
			log.Errorf("Error replacing content in file: %s\n", err)
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
			log.Errorf("Error getting worktree status: %s\n", err)
			return
		}

		for path, s := range st {
			if s.Worktree == git.Modified {
				sz, ok := sizeMemo[path]
				if !ok {
					log.Debugf("%c %c  %s\n", rune(s.Staging), rune(s.Worktree), path)
				}

				abs := filepath.Join(repoPath, path)
				info, err := os.Stat(abs)
				if err != nil {
					log.Errorf("Error getting file info: %s\n", err)
					continue
				}
				nz := info.Size()
				if nz != sz {
					sizeMemo[path] = info.Size()
					log.Infof("***  %s\n", path)
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
