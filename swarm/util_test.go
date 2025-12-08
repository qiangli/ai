package swarm

import (
	"testing"
)

func TestActionNameFromFile(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"agents/pack/agent.yaml", "pack"},
		{"agents/pack/pack.yaml", "pack"},
		{"agents/pack/sub.yaml", "pack/sub"},
		{"tools/kit/name.yaml", "kit:name"},
		{"invalid/path.yaml", ""},
	}

	for _, tc := range tests {
		got := ActionNameFromFile(tc.file)
		if got != tc.want {
			t.Errorf("nameFromFile(%q) = %q; want %q", tc.file, got, tc.want)
		}
	}
}
