package util

import (
	_ "embed"
	"fmt"
	"testing"
)

func TestRender(t *testing.T) {
	txt := Render(testContent)
	fmt.Println(txt)
}

//go:embed testdata/glamour.md
var testContent string
