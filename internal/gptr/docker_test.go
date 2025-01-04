package gptr

import (
	"context"
	"testing"
)

func TestBuildRunGptr(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tests := []struct {
		name  string
		query string
	}{
		{"Test", "Renewable energy sources and their potential"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := BuildGPTRImage(ctx); err != nil {
				t.Errorf("BuildGPTRImage() error = %v", err)
				return
			}
			if err := RunGPTRContainer(ctx, tt.query, "/tmp/gptr"); err != nil {
				t.Errorf("RunGPTRContainer() error = %v", err)
				return
			}
		})
	}
}
