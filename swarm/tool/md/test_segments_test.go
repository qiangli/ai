package md

import (
	"fmt"
	"testing"

	"github.com/yuin/goldmark"
	text "github.com/yuin/goldmark/text"
)

func TestSegments(t *testing.T) {
	source := `### Build

---
dependencies:
  - tidy
---`

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))

	fmt.Println("=== Node walk ===")
	node := doc.FirstChild()
	for node != nil {
		fmt.Printf("%T: ", node)
		if lines := node.Lines(); lines != nil && lines.Len() > 0 {
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				fmt.Printf("  [%d:%d] %q", seg.Start, seg.Stop, string(seg.Value(sourceBytes)))
			}
		} else {
			fmt.Printf("(no lines)")
		}
		fmt.Println()
		node = node.NextSibling()
	}
}
