package internal

import (
	"testing"
)

func TestParse(t *testing.T) {
	data, err := ReadFile("testdata/pg.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := ParseMarkdown(data)
	if doc.CodeBlocks[0].Language != "bash" {
		t.Errorf("Expected 'bash', got %s", doc.CodeBlocks[0].Language)
	}
}
