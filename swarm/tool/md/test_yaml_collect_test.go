package md

import (
	"fmt"
	"testing"

	"github.com/qiangli/ai/swarm/api"
	"github.com/yuin/goldmark"
	ast "github.com/yuin/goldmark/ast"
	text "github.com/yuin/goldmark/text"
)

func TestYAMLCollect(t *testing.T) {
	source := `### Build

Build the project.

---
dependencies:
  - tidy
  - test
arguments:
  env: production
  retries: "3"
---

` + "```bash\necho build\n```"

	sourceBytes := []byte(source)
	parser := goldmark.New()
	doc := parser.Parser().Parse(text.NewReader(sourceBytes))

	// Find first thematic break
	var firstBreak ast.Node
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			if _, ok := n.(*ast.ThematicBreak); ok {
				firstBreak = n
				return ast.WalkStop, nil
			}
		}
		return ast.WalkContinue, nil
	})

	if firstBreak == nil {
		t.Fatal("no thematic break found")
	}

	yamlText, nextNode := collectYAMLBetweenBreaks(firstBreak, sourceBytes)
	fmt.Printf("YAML text:\n%s\n", yamlText)
	fmt.Printf("Next node: %T\n", nextNode)

	// Try to parse it
	task := &api.Task{Name: "test"}
	err := parseTaskYAML([]byte(yamlText), task)
	if err != nil {
		t.Fatalf("parse YAML error: %v", err)
	}
	fmt.Printf("Dependencies: %d\n", len(task.Dependencies))
	for i, dep := range task.Dependencies {
		fmt.Printf("  Dep %d: %s\n", i, dep.Name)
	}
	fmt.Printf("Arguments: %v\n", task.Arguments)
}
