package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
)

func main() {
	// discard bin /ai
	args := os.Args[1:]

	//
	if len(args) == 0 {
		// no args, show help
		args = []string{"/help:help"}
	} else {
		// support execution of ai script file (.sh or .yaml)
		// shebang
		// #!/usr/bin/env ai ACTION --script
		// if the first arg is a file (after the ai bin), action has not been specified in the script file.
		ext := filepath.Ext(args[0])
		switch ext {
		case ".sh", ".bash":
			args = append([]string{"/sh:bash", "--script", "file:///" + args[0]}, args[1:]...)
		case ".yaml", ".yml":
			// TODO supprot main entry detection
			// for now. it is required to specify the action.
			internal.Exit(fmt.Errorf("Failed to run. action required. ex. #!/usr/bin/env ai ACTION --script"))
		case ".txt", ".md", ".markdown", "json", "jsonc":
			args = append([]string{"/ai:call_llm", "--content", "file:///" + args[0]}, args[1:]...)
		default:
			// ok
		}
	}

	if err := agent.Run(args); err != nil {
		internal.Exit(err)
	}
}
