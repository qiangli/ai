package ddg

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()

	cli := New(1, "test")
	if cli == nil {
		t.Error("expected cli not to be nil")
	}
	cli.userAgent = SafariUserAgent
	cli.maxResults = 5

	query := "Help me plan an adventure to California"
	ctx := context.Background()
	result, err := cli.Search(ctx, query)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	t.Logf("result: %+v", result)
}
