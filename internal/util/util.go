package util

import (
	"os"
	"runtime"
)

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
