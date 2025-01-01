//go:build darwin
// +build darwin

package util

import (
	"bytes"
	"fmt"
	"os/exec"
)

func GetMemoryChipInfo() (string, error) {
	cmd := exec.Command("system_profiler", "SPMemoryDataType")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to execute system_profiler: %v", err)
	}
	return out.String(), nil
}
