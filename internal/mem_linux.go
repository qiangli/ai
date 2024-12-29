//go:build linux
// +build linux

package internal

import (
	"bytes"
	"fmt"
	"os/exec"
)

func GetMemoryChipInfo() (string, error) {
	cmd := exec.Command("dmidecode", "-t", "memory")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute dmidecode: %v", err)
	}
	return out.String(), nil
}
