//go:build windows
// +build windows

package util

import (
	"bytes"
	"fmt"
	"os/exec"
)

func GetMemoryChipInfo() (string, error) {
	cmd := exec.Command("powershell", "Get-WmiObject", "Win32_PhysicalMemory")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute PowerShell command: %v", err)
	}
	return out.String(), nil
}
