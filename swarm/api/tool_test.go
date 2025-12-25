package api

import (
	"testing"
)

func TestKitname(t *testing.T) {
	tests := []struct {
		kit      string
		name     string
		expected string
	}{
		{"agent", "ed", "agent__ed__ed"},
	}
	for _, tt := range tests {
		tid := NewKitname(tt.kit, tt.name).ID()
		if tid != tt.expected {
			t.Fatalf("expeted: %s got %s", tt.expected, tid)
		}
	}
}
