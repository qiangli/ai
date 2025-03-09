package agent

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaptinlin/jsonrepair"
)

func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}

	ip := net.ParseIP(host)

	if ip != nil && ip.IsLoopback() {
		return true
	}

	if host == "localhost" {
		return true
	}

	return false
}

// clipText truncates the input text to no more than the specified maximum length.
func clipText(text string, maxLen int) string {
	if len(text) > maxLen {
		return strings.TrimSpace(text[:maxLen]) + "\n[more...]"
	}
	return text
}

// baseCommand returns the last part of the string separated by /.
func baseCommand(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	sa := strings.Split(s, "/")
	return sa[len(sa)-1]
}

// tryUnmarshal tries to unmarshal the data into the v.
// If it fails, it will try to repair the data and unmarshal it again.
func tryUnmarshal(data string, v any) error {
	err := json.Unmarshal([]byte(data), v)
	if err == nil {
		return nil
	}

	repaired, err := jsonrepair.JSONRepair(data)
	if err != nil {
		return fmt.Errorf("failed to repair JSON: %v", err)
	}
	return json.Unmarshal([]byte(repaired), v)
}

func resolveWorkspaceBase(workspace string) (string, error) {
	workspace, err := validatePath(workspace)
	if err != nil {
		return "", err
	}

	// check if the workspace path or any of its parent contains a git repository
	// if so, use the git repository as the workspace
	if workspace, err = detectGitRepo(workspace); err != nil {
		return "", err
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
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
			return path, nil
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	return path, nil
}

func detectGitRepo(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	original := path
	for {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			return path, nil
		}
		np := filepath.Dir(path)
		if np == path || np == "/" {
			break
		}
		path = np
	}
	return original, nil
}

// listRoots returns a list of root directories.
// It includes the current working directory (or parent for git repo) and the temporary directory.
func listRoots() ([]string, error) {
	var roots []string
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	ws, err := resolveWorkspaceBase(wd)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace: %w", err)
	}
	roots = append(roots, ws)

	tempDir := os.TempDir()
	if tempDir == "" {
		return nil, fmt.Errorf("failed to get temporary directory")
	}
	roots = append(roots, tempDir)

	return roots, nil
}
