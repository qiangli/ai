//go:generate go run main.go

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// const promptsDir = "prompts"

func GeneratePrompts() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	dir := filepath.Dir(filename)
	generatePrompts(dir)
}

const generated = `//DO NOT EDIT. This file is generated.
package resource

import _ "embed"

%s

var Prompts = map[string]string{
%s
}
`

func generatePrompts(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		os.Exit(1)
	}

	parent := filepath.Dir(dir)
	base := filepath.Base(dir)

	outputFile, err := os.Create(filepath.Join(parent, "generated.go"))
	if err != nil {
		fmt.Println("Error creating prompts file:", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	var embeds []string
	var entries []string

	for _, file := range files {
		if !file.IsDir() {
			fileName := file.Name()
			if filepath.Ext(fileName) == ".go" {
				continue
			}
			fileBase := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			embeds = append(embeds, fmt.Sprintf("//go:embed %s/%s", base, fileName))
			embeds = append(embeds, fmt.Sprintf("var %s string", fileBase))
			embeds = append(embeds, "")

			entries = append(entries, fmt.Sprintf("\t\"%s\": %s,", fileBase, fileBase))
		}
	}

	fmt.Fprintln(outputFile, fmt.Sprintf(generated, strings.Join(embeds, "\n"), strings.Join(entries, "\n")))
}

// import (
// 	"flag"
// 	"fmt"

// 	"github.com/qiangli/ai/internal/agent/resource"
// )

func main() {
	// flag.Parse()
	GeneratePrompts()
	fmt.Println("Prompt resource mapping generated")
}
