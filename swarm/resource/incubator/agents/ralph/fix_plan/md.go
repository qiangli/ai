// --mime-type=text/x-go-template
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var baseDir = "{{.base_dir}}"

const content = "{{asset \"/templates/fix_plan.md\"}}"

func main() {
	if baseDir == "" || baseDir == "<no value>" {
		fmt.Fprintln(os.Stderr, "missing required parameter: base_dir")
		os.Exit(2)
	}
	outPath := filepath.Join(baseDir, "@fix_plan.md")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, []byte(content), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println(outPath)
}
