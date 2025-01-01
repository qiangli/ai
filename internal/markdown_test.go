package internal

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	data, err := os.ReadFile("testdata/psql.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := ParseMarkdown(string(data))
	if doc.CodeBlocks[0].Language != "bash" {
		t.Errorf("Expected 'bash', got %s", doc.CodeBlocks[0].Language)
	}
}
