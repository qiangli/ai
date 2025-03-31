package util

import (
	"os"
	"path/filepath"
	"strings"
)

// ListCommands returns the full path of the first valid executable command encountered in the PATH
// environment variable.
func ListCommands() map[string]string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return map[string]string{}
	}

	uniq := make(map[string]string) // command name -> full path
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, pathDir := range paths {
		files, err := os.ReadDir(pathDir)
		if err != nil {
			continue
		}
		for _, file := range files {
			name := file.Name()
			fullPath := filepath.Join(pathDir, name)

			// Check if the file is executable and the command hasn't been registered yet
			if !file.IsDir() && IsExecutable(fullPath) {
				if _, exists := uniq[name]; !exists {
					uniq[name] = fullPath
				}
			}
		}
	}

	return uniq
}

func IsExecutable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode.IsRegular() && mode&0111 != 0
}
