// --mime-type=text/x-go-template
package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func saveAgentMD(baseDir, content string) {
	if baseDir == "" || baseDir == "<no value>" {
		fmt.Fprintln(os.Stderr, "missing required parameter: base_dir")
		os.Exit(2)
	}
	outPath := filepath.Join(baseDir, "@AGENT.md")
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

func main() {
	var baseDir = "{{.base_dir}}"
	var content = "{{asset \"templates/AGENT.md\"}}"
	saveAgentMD(baseDir, content)
}
