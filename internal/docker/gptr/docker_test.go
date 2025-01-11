package gptr

import (
	"context"
	"testing"
)

func TestGenerateReport(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tests := []struct {
		name  string
		query string
		out   string
	}{
		{"Test", "Renewable energy sources and their potential", "/tmp/gptr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := GenerateReport(ctx, tt.query, tt.out)
			if err != nil {
				t.Errorf("GenerateReport error = %v", err)
				return
			}
		})
	}
}
