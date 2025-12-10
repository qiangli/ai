package main

import (
	"context"
	"os"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

func main() {
	ctx := context.TODO()

	args := os.Args

	// support execution of ai script file (.sh or .yaml)
	shebang := strings.HasSuffix(args[0], ".yaml") || strings.HasSuffix(args[0], ".sh")

	if shebang {
		args = append([]string{"/sh:bash", "--script", args[0]}, args[1:]...)
	} else {
		if len(args) <= 1 {
			// $ai 
			// show help 
			args = []string{"/help:help"}
		} else {
			//  $ai args...
			// discard the command itself
			args = args[1:]
		}
	}

	if err := agent.Run(ctx, args); err != nil {
		internal.Exit(ctx, err)
	}
}
