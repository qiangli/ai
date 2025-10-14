package conf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestNewBlobs(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.FailNow()
	}
	app := &api.AppConfig{
		Base: filepath.Join(home, ".ai"),
	}
	nb, err := NewBlobs(app, "")
	if err != nil {
		t.FailNow()
	}
	blob := &api.Blob{
		ID:       "test",
		MimeType: "text/plain",
		Content:  []byte("this is a test"),
	}

	if err = nb.Put(blob.ID, blob); err != nil {
		t.FailNow()
	}
	read, err := nb.Get(blob.ID)
	if err != nil {
		t.FailNow()
	}
	if string(read.Content) != string(blob.Content) {
		t.Fatalf("put/get not same")
	}
}
