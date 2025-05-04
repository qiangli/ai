package shell

import (
	"sort"
	"testing"
)

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name   string
		input1 string
		input2 string
	}{
		{"home expansion", "~/.bashrc", "${HOME}/.bashrc"},
		{"* expansion", "explore/etc/*", "explore/etc/icons"},
		// golang builtin glob fails to glob input1
		{"[] expansion", "explore/img/{[1-9],[1-3][0-9],4[0-4]}.png", "explore/img/*.png"},
		{"[] expansion", "explore/img/*.p?g", "explore/img/*.png"},
		{"non expansion", "explore/img/1.png", "explore/img/1.png"},
		// both returns nil
		{"any string", "xxx", "xxx"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result1, err := expandPath(test.input1)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			result2, err := expandPath(test.input2)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if len(result1) != len(result2) {
				t.Errorf("expected %d results, got %d", len(result2), len(result1))
			}
			sort.Strings(result1)
			sort.Strings(result2)
			for i := 0; i < len(result1); i++ {
				if result1[i] != result2[i] {
					t.Errorf("expected %q, got %q", result2[i], result1[i])
				}
			}
		})
	}
}
