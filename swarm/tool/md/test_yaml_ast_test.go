package md

import (
	"fmt"
	"testing"

	"github.com/yuin/goldmark"
	text "github.com/yuin/goldmark/text"
)

func TestYAMLAST(t *testing.T) {
	source := `---
dependencies:
  - tidy
  - test
arguments:
  env: production
  retries: "3"
---`

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))

	fmt.Println("=== YAML Block AST ===")
	printNode(doc, sourceBytes, 0)
}
