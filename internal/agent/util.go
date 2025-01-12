package agent

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

// ValidatePath returns the absolute path of the given path.
// If the path is empty, it returns an error. If the path is not an absolute path,
// it converts it to an absolute path.
// If the path does not exist, it creates the path and returns the absolute path.
// If the path exists, it returns its absolute path.
func ValidatePath(path string) (string, error) {
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
