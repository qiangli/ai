package watch

import (
	"testing"
)

func TestWatchGit(t *testing.T) {
	// err := WatchGit("/tmp/ws")

	err := WatchGit("../../../ai")
	if err != nil {
		t.Errorf("Error watching git repository: %v", err)
	}
}
