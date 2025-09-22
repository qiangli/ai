package shell

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	git "gopkg.in/src-d/go-git.v4"

	"github.com/qiangli/ai/swarm/log"
)

const (
	// HOME
	userHomeEnv = "HOME"
	// GIT_ROOT
	gitRootEnv = "GIT_ROOT"
	// // WORKSPACE
	// workspaceEnv = "WORKSPACE"
)

func createPrompter(ctx context.Context) (func(), error) {
	const app = "ai"
	const ps = ">"
	const dirPs = "\033[0;35m%s\033[0;34m\033[0;36m/%s \033[0;32m%s\033[0;34m%s \033[0m"
	const repoPs = "\033[0;35m%s@\033[0;33m%s\033[0;34m\033[0;36m/%s \033[0;32m%s\033[0;34m%s \033[0m"

	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	hostname = strings.SplitN(hostname, ".", 2)[0]

	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	home := u.HomeDir
	username := u.Username

	regex, err := regexp.Compile(`https://.*/.*/(.*)(\\.git)?`)
	if err != nil {
		return nil, err
	}

	// set HOME
	os.Setenv(userHomeEnv, home)

	return func() {
		where := hostname
		dir, _ := os.Getwd()

		// default
		var baseDir string
		baseDir = filepath.Base(dir)

		log.GetLogger(ctx).Debug("dir: %s %s\n", where, dir)
		log.GetLogger(ctx).Debug("baseDir: %s\n", baseDir)

		// relative to home
		if strings.HasPrefix(dir, home) {
			where = username
			baseDir, _ = filepath.Rel(home, dir)
			log.GetLogger(ctx).Debug("home: %s\n", home)
			log.GetLogger(ctx).Debug("user baseDir: %s\n", baseDir)
		}

		repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
		if err != nil {
			fmt.Printf(dirPs, where, baseDir, app, ps)
			return
		}

		// git is init-ed but no remote
		list, _ := repo.Remotes()
		if len(list) == 0 {
			where = ""
		}
		for _, remote := range list {
			if remote.Config().Name == "origin" {
				url := remote.Config().URLs[0]
				if regex.MatchString(url) {
					where = regex.FindStringSubmatch(url)[1]
					break
				}
			}
		}

		// relative to git repo base if in git repo
		worktree, err := repo.Worktree()
		if err == nil {
			repoDir := worktree.Filesystem.Root()

			// set GIT_ROOT
			os.Setenv(gitRootEnv, repoDir)

			if strings.HasPrefix(dir, repoDir) {
				baseDir, _ = filepath.Rel(repoDir, dir)
				log.GetLogger(ctx).Debug("repoDir: %s\n", repoDir)
				log.GetLogger(ctx).Debug("repo baseDir: %s\n", baseDir)
			}
		}

		rawhead, _ := repo.Head()
		head1 := strings.Split(rawhead.String(), "/")
		head := head1[len(head1)-1]

		log.GetLogger(ctx).Debug("repoName: %s\n", where)
		log.GetLogger(ctx).Debug("head: %s\n", head)
		log.GetLogger(ctx).Debug("baseDir: %s\n", baseDir)

		shortName := shortName(head)
		log.GetLogger(ctx).Debug("shortName: %s\n", shortName)

		fmt.Printf(repoPs, where, shortName, baseDir, app, ps)
	}, nil
}

func isValidHash(hash string) bool {
	if len(hash) != 40 {
		return false
	}
	match, _ := regexp.MatchString("^[0-9a-f]+$", hash)
	return match
}

// return short hash if valid, otherwise return the original value
func shortName(s string) string {
	parts := strings.Split(s, " ")
	for i, part := range parts {
		if isValidHash(part) {
			parts[i] = part[:7]
		}
	}
	return strings.Join(parts, " ")
}
