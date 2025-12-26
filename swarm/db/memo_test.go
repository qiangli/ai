package db

import (
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/qiangli/ai/swarm/api"
)

func TestDBLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	home, _ := os.UserHomeDir()
	mem, _ := OpenMemoryStore(filepath.Join(home, "/.ai"), "test.db")
	opt := &api.MemOption{
		MaxSpan:    14400,
		Roles:      []string{"user", "assistant"},
		MaxHistory: 100,
		Offset:     0,
	}

	messages, _ := mem.Load(opt)
	t.Logf("%v\n", messages)
}
