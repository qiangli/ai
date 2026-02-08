package md

import (
	"fmt"
	"os"
	"testing"
)

func TestDebugSimple(t *testing.T) {
	c, err := os.ReadFile("./testdata/fixtures/simple.md")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := string(c)
	fmt.Println("raw content:\n", lines)
	tf, err := Parse(lines)
	if err != nil {
		fmt.Printf("Parse err: %v\n", err)
		return
	}
	fmt.Printf("Parsed ok: title=%q groups=%d\n", tf.Title, len(tf.Tasks))
}
