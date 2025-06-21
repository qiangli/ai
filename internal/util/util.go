package util

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
