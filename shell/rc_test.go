package shell

import (
	"testing"
)

func TestAlias(t *testing.T) {
	tests := []struct {
		bin string
	}{
		{bin: "/bin/bash"},
		{bin: "/bin/sh"},
	}
	for _, test := range tests {
		t.Run(test.bin, func(t *testing.T) {
			aliases, err := listAlias(test.bin)
			if err != nil {
				t.Fatalf("Failed to execute alias command: %v", err)
			}
			t.Logf("Aliases: %v", aliases)
		})
	}
}

func TestEnv(t *testing.T) {
	tests := []struct {
		bin string
	}{
		{bin: "/bin/bash"},
		{bin: "/bin/sh"},
	}
	for _, test := range tests {
		t.Run(test.bin, func(t *testing.T) {
			envs, err := env(test.bin)
			if err != nil {
				t.Fatalf("Failed to execute env command: %v", err)
			}
			t.Logf("Envs: %v", envs)
		})
	}
}
