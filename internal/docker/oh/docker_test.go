package oh

import (
	"context"
	"testing"
)

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tests := []struct {
		name  string
		query string
	}{
		{"Test", "write a bash script that prints hi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := Run(ctx, tt.query)
			if err != nil {
				t.Errorf("Run error = %v", err)
				return
			}
		})
	}
}
