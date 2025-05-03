package util

import (
	"os"
	"testing"
)

func TestGetShellInfo(t *testing.T) {
	shells := []struct {
		env       string
		name      string
		wantName  string
		wantError bool
	}{
		{"/bin/bash", "bash", "bash", false},
		{"/bin/zsh", "zsh", "zsh", false},
		{"/bin/csh", "csh", "csh", false},
		{"/bin/foolish", "foolish", "", true},
		{"", "", "", true},
	}

	for _, tc := range shells {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("SHELL", tc.env)
			} else {
				os.Unsetenv("SHELL")
			}
			info, err := GetShellInfo()
			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Did not expect error, got: %v", err)
				}
				if info["Name"] != tc.wantName {
					t.Errorf("Expected Name %s, got %s", tc.wantName, info["Name"])
				}
				if info["Path"] != tc.env {
					t.Errorf("Expected Path %s, got %s", tc.env, info["Path"])
				}
				if info["Version"] == "" {
					t.Errorf("Expected non-empty Version")
				}
			}
			t.Logf("Shell Info: %+v", info)
		})
	}
}
