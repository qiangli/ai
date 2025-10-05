package internal

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"

	// "sort"
	// "strings"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/log"
)

func getCurrentUser() string {
	currentUser, err := user.Current()
	if err != nil {
		return "unkown"
	}
	return currentUser.Username
}

// func printAIEnv() {
// 	// Get the current environment variables
// 	envs := os.Environ()
// 	var filteredEnvs []string
// 	for _, v := range envs {
// 		if strings.HasPrefix(v, "AI_") {
// 			filteredEnvs = append(filteredEnvs, v)
// 		}
// 	}
// 	sort.Strings(filteredEnvs)
// 	log.GetLogger(ctx).Debugf("AI env: %v\n", filteredEnvs)
// }

var validFormats = []string{"raw", "text", "json", "markdown", "tts"}

func isValidFormat(format string) bool {
	return slices.Contains(validFormats, format)
}

func Validate(app *api.AppConfig) error {
	if app.Format != "" && !isValidFormat(app.Format) {
		return fmt.Errorf("invalid format: %s", app.Format)
	}
	return nil
}

// resolveWorkspaceDir returns the workspace directory.
// If the workspace is not provided, it returns the temp dir.
func resolveWorkspaceDir(ws string) (string, error) {
	if ws != "" {
		return ensureWorkspace(ws)
	}
	// ws, err := os.Getwd()
	// if err != nil {
	// 	return "", err
	// }
	// return resolveRepoDir(ws)
	return tempDir()
}

// // resolveRepoDir returns the directory of the current git repository
// func resolveRepoDir(ws string) (string, error) {
// 	if ws == "" {
// 		wd, err := os.Getwd()
// 		if err != nil {
// 			return "", err
// 		}
// 		ws = wd
// 	}
// 	dir, err := detectGitRepo(ws)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to detect git repository: %w", err)
// 	}
// 	return dir, nil
// }

// func homeDir() (string, error) {
// 	return os.UserHomeDir()
// }

func tempDir() (string, error) {
	return os.TempDir(), nil
}

// // detectGitRepo returns the directory of the git repository
// // containing the given path.
// // If the path is not in a git repository, it returns the original path.
// func detectGitRepo(path string) (string, error) {
// 	if path == "" {
// 		return "", fmt.Errorf("path is empty")
// 	}
// 	original := path
// 	for {
// 		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
// 			return path, nil
// 		}
// 		np := filepath.Dir(path)
// 		if np == path || np == "/" {
// 			break
// 		}
// 		path = np
// 	}
// 	return original, nil
// }

func ensureWorkspace(ws string) (string, error) {
	workspace, err := validatePath(ws)
	if err != nil {
		return "", err
	}

	// ensure the workspace directory exists
	if err := os.MkdirAll(workspace, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	return workspace, nil
}

// ValidatePath returns the absolute path of the given path.
// If the path is empty, it returns an error.
// If the path is not an absolute path, it converts it to an absolute path.
// If the path exists, it returns its absolute path.
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		path = absPath
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	return path, nil
}

// func expandWithDefault(input string) string {
// 	return os.Expand(input, func(key string) string {
// 		parts := strings.SplitN(key, ":-", 2)
// 		value := os.Getenv(parts[0])
// 		if value == "" && len(parts) > 1 {
// 			return parts[1]
// 		}
// 		return value
// 	})
// }
