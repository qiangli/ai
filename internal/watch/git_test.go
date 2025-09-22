package watch

import (
	"context"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestWatchRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	ctx := context.TODO()

	err := WatchRepo(ctx, &api.AppConfig{
		Workspace: "../../../ai",
	})

	if err != nil {
		t.Errorf("Error watching git repository: %v", err)
	}
}
