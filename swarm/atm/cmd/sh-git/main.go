package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

	stdout, stderr, err := gitkit.ExecGit("", argv[1:]...)
	if stdout != "" {
		fmt.Fprint(os.Stdout, stdout)
	}
	if stderr != "" {
		fmt.Fprint(os.Stderr, stderr)
	}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
		os.Exit(1)
	}
}

