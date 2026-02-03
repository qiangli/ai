package index_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atm/memory"
)

func TestIndexSearch(t *testing.T) {
	tmp := t.TempDir()
	cfg := memory.MemoryConfig{
		Enabled:    true,
		Provider:   "local",
		Model:      "random",
		StorePath:  tmp,
		ExtraPaths: []string{"*.md"},
		Query:      memory.QueryStruct{MaxResults: 5}, // note: adjust if struct name
	}
	testFile := filepath.Join(tmp, "test.md")
	os.WriteFile(testFile, []byte("# Test\nContent for search test memory semantic."), 0644)
	memory.IndexWorkspace(tmp, cfg)
	res, err := memory.Search("memory semantic", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("no results expected")
	}
	t.Log(res[0])
}