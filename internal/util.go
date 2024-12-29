package internal

import (
	"io"
	"os"
	"runtime"
)

func MaxLen(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func ReadFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func HomeDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
		if homeDir == "" && runtime.GOOS == "windows" {
			homeDir = os.Getenv("USERPROFILE")
		}
	}
	return homeDir
}
