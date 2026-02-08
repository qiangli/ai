package md

import (
	"fmt"
	"testing"

	"github.com/yuin/goldmark"
	ast "github.com/yuin/goldmark/ast"
	text "github.com/yuin/goldmark/text"
)

func TestPositions(t *testing.T) {
	source := `### Build

---
dependencies:
  - tidy
---

` + "```bash\necho\n```"

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))

	// Find thematic breaks
	var breaks []*ast.ThematicBreak
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if br, ok := n.(*ast.ThematicBreak); ok {
				breaks = append(breaks, br)
			}
		}
		return ast.WalkContinue, nil
	})

	fmt.Printf("Found %d thematic breaks\n", len(breaks))
	for i, br := range breaks {
		lines := br.Lines()
		fmt.Printf("Break %d: lines=%d\n", i, lines.Len())
		for j := 0; j < lines.Len(); j++ {
			seg := lines.At(j)
			fmt.Printf("  Line %d: start=%d stop=%d value=%q\n", j, seg.Start, seg.Stop, string(seg.Value(sourceBytes)))
		}
	}

	if len(breaks) >= 2 {
		startLines := breaks[0].Lines()
		endLines := breaks[1].Lines()
		if startLines != nil && endLines != nil && startLines.Len() > 0 && endLines.Len() > 0 {
			startPos := startLines.At(startLines.Len() - 1).Stop
			endPos := endLines.At(0).Start
			fmt.Printf("\nStart pos: %d, End pos: %d\n", startPos, endPos)
			yamlText := string(sourceBytes[startPos:endPos])
			fmt.Printf("YAML text:\n%s\n", yamlText)
		}
	}
}
