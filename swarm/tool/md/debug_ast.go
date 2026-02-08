package md

import (
	"fmt"
	"github.com/yuin/goldmark"
	ast "github.com/yuin/goldmark/ast"
	text "github.com/yuin/goldmark/text"
)

func DebugAST() {
	source := `### Task Two

Second task with dependencies

---
dependencies:
  - task-one
---

` + "```bash\necho two\n```"

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))

	fmt.Println("=== AST Structure ===")
	printNode(doc, sourceBytes, 0)
}

func printNode(n ast.Node, source []byte, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	nodeType := fmt.Sprintf("%T", n)
	fmt.Printf("%s%s", indent, nodeType)

	// Print node-specific info
	switch v := n.(type) {
	case *ast.Heading:
		fmt.Printf(" level=%d text=%q", v.Level, nodeText(n, source))
	case *ast.Text:
		fmt.Printf(" text=%q", string(v.Segment.Value(source)))
	case *ast.Paragraph:
		fmt.Printf(" text=%q", nodeText(n, source))
	case *ast.FencedCodeBlock:
		lang := string(v.Language(source))
		fmt.Printf(" lang=%q", lang)
	case *ast.ThematicBreak:
		fmt.Printf(" (---)")
	case *ast.List:
		fmt.Printf(" tight=%v", v.IsTight)
	case *ast.ListItem:
		fmt.Printf("")
	}
	fmt.Println()

	// Recursively print children
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		printNode(child, source, depth+1)
	}
}
