package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

func main() {
	ctx := context.TODO()

	// discard ai command
	args := os.Args[1:]

	fmt.Printf("main args[1:]: %v\n", args)

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

	if err := agent.Run(ctx, args); err != nil {
		internal.Exit(ctx, err)
	}
}
