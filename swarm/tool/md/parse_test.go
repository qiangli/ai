package md

import (
	"os"
	"testing"
)

func TestParse(t *testing.T) {
	c, err := os.ReadFile("./testdata/task.md")
	if err != nil {
		t.FailNow()
	}
	Parse(string(c))
}
