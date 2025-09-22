package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/qiangli/ai/internal/util"
)

func TestConfirm(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		name          string
		ps            string
		choices       []string
		defaultChoice string
		choice        string
		expected      string
	}{
		{"Test 1", "Run? [y/N] ", []string{"yes", "no"}, "no", "\n", "no"},
		{"Test 2", "Run? [y/N] ", []string{"yes", "no"}, "no", "y\n", "yes"},
		{"Test 3", "Run? [y/N] ", []string{"yes", "no"}, "no", "n\n", "no"},
		{"Test 4", "Run, edit, copy? [y/e/c/N] ", []string{"yes", "edit", "copy", "no"}, "no", "\n", "no"},
		{"Test 5", "Run, edit, copy? [y/e/c/N] ", []string{"yes", "edit", "copy", "no"}, "no", "y\n", "yes"},
		{"Test 6", "Run, edit, copy? [y/e/c/N] ", []string{"yes", "edit", "copy", "no"}, "no", "e\n", "edit"},
		{"Test 7", "Run, edit, copy? [y/e/c/N] ", []string{"yes", "edit", "copy", "no"}, "no", "c\n", "copy"},
		{"Test 8", "Run, edit, copy? [y/e/c/N] ", []string{"yes", "edit", "copy", "no"}, "no", "n\n", "no"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := util.Confirm(ctx, tc.ps, tc.choices, tc.defaultChoice, strings.NewReader(tc.choice))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if expected != tc.expected {
				t.Fatalf("expected %s, got %s", tc.expected, expected)
			}
		})
	}
}
