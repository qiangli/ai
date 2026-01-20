package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/swarm/atm/conf"
)

func main() {
	if isDebugging(os.Args) {
		fmt.Printf("os args: %v\n", os.Args)
	}

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
		// the first arg must be an action (after the ai bin).
		// error if it has not been specified in the script file and is not a shell script.
		ext := filepath.Ext(args[0])
		switch ext {
		case ".sh", ".bash":
			args = append([]string{"/sh:bash", "--script", args[0]}, args[1:]...)
		default:
			if !conf.IsAction(args[0]) {
				// default to @root
				args = []string{"/agent:root/root", "--message", strings.Join(args, " ")}
				// system command
				// args = []string{"/sh:exec", "--command", strings.Join(args, " ")}
				// internal.Exit(fmt.Errorf("Failed to run. action required. ex. #!/usr/bin/env ai ACTION --script"))
			}
		}
	}

	if isDebugging(args) {
		fmt.Printf("args: %v\n", args)
	}

	if err := agent.Run(args); err != nil {
		internal.Exit(err)
	}
}

func isDebugging(args []string) bool {
	for _, v := range args {
		if v == "--verbose" || v == "-verbose" {
			return true
		}
	}
	return false
}
