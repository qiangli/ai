package util

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

type MarkdownDoc struct {
	Title      string
	Contents   []string
	CodeBlocks []CodeBlock
}

type CodeBlock struct {
	Language string
	Code     string
}

func ParseMarkdown(markdownContent string) *MarkdownDoc {
	md := goldmark.New()

	reader := text.NewReader([]byte(markdownContent))
	node := md.Parser().Parse(reader)

	var doc MarkdownDoc

	var buf []string
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if heading, ok := n.(*ast.Heading); ok && entering && heading.Level == 1 {
			doc.Title = string(heading.Text(reader.Source()))
		} else if text, ok := n.(*ast.Text); ok && entering {
			buf = append(buf, string(text.Segment.Value(reader.Source())))
		} else if codeBlock, ok := n.(*ast.FencedCodeBlock); ok && entering {
			language := string(codeBlock.Language(reader.Source()))
			code := string(codeBlock.Text(reader.Source()))
			code = strings.TrimSpace(code)
			if code != "" && language != "" {
				doc.CodeBlocks = append(doc.CodeBlocks, CodeBlock{
					Language: language,
					Code:     code,
				})
			}
		}
		return ast.WalkContinue, nil
	})

	doc.Contents = buf
	return &doc
}
