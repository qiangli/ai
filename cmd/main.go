package main

import (
	"context"
	"os"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/swarm/api"
)

func main() {
	ctx := context.TODO()

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

	// shebang
	// TODO load args from first line of
	var app = &api.AppConfig{}
	app.Arguments = make(map[string]any)
	err := agent.SetupAppConfig(app)
	if err != nil {
		internal.Exit(ctx, err)
	}

	// read args[0] file and parse first line for args

	if err := agent.Run(ctx, app, args); err != nil {
		internal.Exit(ctx, err)
	}
}
