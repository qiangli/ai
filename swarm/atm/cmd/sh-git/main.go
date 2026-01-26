package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/qiangli/ai/swarm/atm/gitkit"
)

func main() {
	// Accept either:
	//   sh-git git status
	// or:
	//   sh-git "git status"
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: sh-git <git ...> | sh-git \"git ...\"")
		os.Exit(2)
	}

	var argv []string
	if len(os.Args) == 2 {
		// Single string; split on whitespace (best-effort; recommend passing args).
		argv = strings.Fields(os.Args[1])
	} else {
		argv = os.Args[1:]
	}

	if len(argv) == 0 || argv[0] != "git" {
		fmt.Fprintln(os.Stderr, "error: command must start with 'git'")
		os.Exit(2)
	}

	stdout, stderr, exitCode, err := gitkit.RunGitExitCode("", argv[1:]...)
	if stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	if err != nil {
		// If run returned an explicit exit code, use it; otherwise default to 1.
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		os.Exit(1)
	}
}
