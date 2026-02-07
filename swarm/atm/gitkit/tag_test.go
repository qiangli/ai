package gitkit

import (
	"github.com/go-git/go-git/v5"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestTagsListAndCreate(t *testing.T) {
	// create a temp repo
	dir, err := ioutil.TempDir("", "gitkittest")
	if err != nil {
		t.Fatalf("tempdir: %v", err)
	}
	defer os.RemoveAll(dir)
	r, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	// create an initial commit so we can tag
	fpath := filepath.Join(dir, "README.md")
	if err := os.WriteFile(fpath, []byte("hello"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	w, err := r.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}
	if _, err := w.Add("README.md"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, _, _, err := Commit(dir, "init", nil); err != nil {
		t.Fatalf("commit: %v", err)
	}
	// ensure no tags initially
	list, err := Tags(nil, r)
	if err != nil {
		t.Fatalf("tags list: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 tags, got %v", list)
	}
	// create a tag via Tag function
	if _, _, err := Tag(dir, "v1.0", "HEAD", false, ""); err != nil {
		t.Fatalf("create tag: %v", err)
	}
	list2, err := Tags(nil, r)
	if err != nil {
		t.Fatalf("tags list after: %v", err)
	}
	found := false
	for _, v := range list2 {
		if v == "v1.0" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tag v1.0 not found in %v", list2)
	}
}
