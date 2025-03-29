package watch

import (
	"testing"

	"github.com/qiangli/ai/internal"
)

func TestWatchRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	err := WatchRepo(&internal.AppConfig{
		Workspace: "../../../ai",
	})

	if err != nil {
		t.Errorf("Error watching git repository: %v", err)
	}
}
