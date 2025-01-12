package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// ListCommands returns the full path of the first valid executable command encountered in the PATH
// environment variable. If nameOnly is true, it returns only the command name.
func ListCommands(nameOnly bool) ([]string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return nil, errors.New("PATH environment variable is not set")
	}

	uniqueCommands := make(map[string]string) // command name -> full path
	paths := strings.Split(pathEnv, string(os.PathListSeparator))

	for _, pathDir := range paths {
		files, err := os.ReadDir(pathDir)
		if err != nil {
			continue
		}

		for _, file := range files {
			commandName := file.Name()
			fullPath := filepath.Join(pathDir, commandName)

			// Check if the file is executable and the command hasn't been registered yet
			if !file.IsDir() && IsExecutable(fullPath) {
				if _, exists := uniqueCommands[commandName]; !exists {
					uniqueCommands[commandName] = fullPath
				}
			}
		}
	}

	commands := make([]string, 0, len(uniqueCommands))
	for name, fullPath := range uniqueCommands {
		if nameOnly {
			commands = append(commands, name)
			continue
		}
		commands = append(commands, fullPath)
	}

	return commands, nil
}

func IsExecutable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode.IsRegular() && mode&0111 != 0
}
