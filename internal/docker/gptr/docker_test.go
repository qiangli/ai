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
		name       string
		reportType string
		tone       string
		query      string
		out        string
	}{
		{"Test", "detailed_report", "humorous", "Renewable energy sources and their potential", "/tmp/gptr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := GenerateReport(ctx, tt.reportType, tt.tone, tt.query, tt.out)
			if err != nil {
				t.Errorf("GenerateReport error = %v", err)
				return
			}
		})
	}
}
