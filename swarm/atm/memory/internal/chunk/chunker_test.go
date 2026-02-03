package chunk

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestChunksFromLines(t *testing.T) {
	lines := []string{
		"L1",
		"L2 long line with words to count chars Lorem ipsum dolor.",
		"L3",
		"L4",
		"L5",
		"L6 long long long line to force new chunk Lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod.",
		"L7",
		"L8",
		"L9",
		"L10",
		"L11",
	}
	chunks := ChunksFromLines("/test.md", lines)
	if len(chunks) < 1 {
		t.Fatal("no chunks produced")
	}
	if chunks[0].StartLine != 1 {
		t.Errorf("first chunk start line expected 1 got %d", chunks[0].StartLine)
	}
	if len(chunks) > 1 {
		if chunks[1].StartLine >= chunks[0].EndLine {
			t.Error("no overlap")
		}
	}
}