package main

import (
	"os"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

func main() {
	// discard ai command bin .../ai
	args := os.Args[1:]

	if len(args) == 0 {
		// no args, show help
		args = []string{"/help:help"}
	} else {
		// support execution of ai script file (.sh or .yaml)
		shebang := (strings.HasSuffix(args[0], ".yaml") || strings.HasSuffix(args[0], ".sh"))
		if shebang {
			args = append([]string{"/sh:bash", "--script", args[0]}, args[1:]...)
		}
	}

	if err := agent.Run(args); err != nil {
		internal.Exit(err)
	}
}
