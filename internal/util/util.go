package util

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// func HomeDir() string {
// 	home, _ := os.UserHomeDir()
// 	if home == "" && runtime.GOOS != "windows" {
// 		if u, err := user.Current(); err == nil {
// 			return u.HomeDir
// 		}
// 	}
// 	return home
// }

func GetEnvVarNames() string {
	names := []string{}
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			names = append(names, pair[0])
		}
	}
	sort.Strings(names)
	return strings.Join(names, "\n")
}

func DetectContentType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer)
	if contentType != "application/octet-stream" {
		return contentType, err
	}

	ext := filepath.Ext(filePath)
	if ext != "" {
		extType := mime.TypeByExtension(ext)
		if extType != "" {
			return extType, nil
		}
	}

	return "application/octet-stream", nil
}

// func DetectGitRepo(path string) (string, error) {
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

// func ResolveWorkspace(workspace string) (string, error) {
// 	workspace, err := ValidatePath(workspace)
// 	if err != nil {
// 		return "", err
// 	}

// 	// ensure the workspace directory exists
// 	if err := os.MkdirAll(workspace, os.ModePerm); err != nil {
// 		return "", fmt.Errorf("failed to create directory: %w", err)
// 	}

// 	return workspace, nil
// }

// // ValidatePath returns the absolute path of the given path.
// // If the path is empty, it returns an error.
// // If the path is not an absolute path, it converts it to an absolute path.
// // If the path exists, it returns its absolute path.
// func ValidatePath(path string) (string, error) {
// 	if path == "" {
// 		return "", fmt.Errorf("path is empty")
// 	}

// 	if !filepath.IsAbs(path) {
// 		absPath, err := filepath.Abs(path)
// 		if err != nil {
// 			return "", fmt.Errorf("failed to get absolute path: %w", err)
// 		}
// 		path = absPath
// 	}
// 	if _, err := os.Stat(path); err != nil {
// 		if os.IsNotExist(err) {
// 			return path, nil
// 		}
// 		return "", fmt.Errorf("failed to stat path: %w", err)
// 	}

// 	return path, nil
// }
