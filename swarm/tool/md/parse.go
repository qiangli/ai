package md

import (
	// "bytes"
	"fmt"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-meta"
	// "github.com/yuin/goldmark/extension"
	// "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"

	"github.com/qiangli/ai/swarm/api"

)

// Parse source into Taskfile
func Parse(source string) {
	var taskfile api.TaskFile
	
	markdown := goldmark.New(
		goldmark.WithExtensions(
			meta.New(
				meta.WithStoresInDocument(),
			),
		),
	)

	doc := markdown.Parser().Parse(text.NewReader([]byte(source)))
	metaData := doc.OwnerDocument().Meta()
	title := metaData["Title"]
	fmt.Print(title)
}
