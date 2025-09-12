package ddg

import (
	"context"
	"testing"
)

func TestSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()

	maxResults := 3

	cli := New(maxResults)
	if cli == nil {
		t.Error("expected cli not to be nil")
	}

	query := "Help me plan an adventure to California"
	ctx := context.Background()
	result, err := cli.Search(ctx, query)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("result: %s", result)
}
