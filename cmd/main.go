package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

func main() {
	args := os.Args

	if len(args) == 1 {
		// no args, show help
		args = []string{"/help:help"}
	} else {
		// support execution of ai script file (.sh or .yaml)
		shebang := (strings.HasSuffix(args[0], ".yaml") || strings.HasSuffix(args[0], ".sh"))
		if shebang {
			args = append([]string{"/sh:bash", "--script", args[0]}, args[1:]...)
		} else {
			// discard ai command bin <...>/ai
			args = args[1:]
		}
	}

	fmt.Printf("%v\n", args)
	if err := agent.Run(args); err != nil {
		internal.Exit(err)
	}
}
