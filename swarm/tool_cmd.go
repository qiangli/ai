package swarm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/internal/bubble/confirm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/vfs"
	"github.com/qiangli/ai/swarm/vos"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os

var _fs vfs.FileSystem = &vfs.VirtualFS{}

// runCommand executes a shell command with args and returns the output
func runCommand(ctx context.Context, command string, args []string) (string, error) {
	log.GetLogger(ctx).Debugf("🏃 %s (%d) %+v\n", command, len(args), args)

	var out []byte
	var err error
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		out, err = _exec.Command("sh", "-c", command).CombinedOutput()
	} else {
		out, err = _exec.Command(command, args...).CombinedOutput()
	}
	if err != nil {
		log.GetLogger(ctx).Errorf("\033[31m✗\033[0m %s: %+v\n", command, err)
		return "", fmt.Errorf("%v\n%s", err, clip(string(out), 500))
	}

	log.GetLogger(ctx).Debugf("🎉 %s: %s\n", command, out)
	return string(out), nil
}

// runCommandVerbose executes a shell command with arguments,
// prints stdout/stderr in real-time, and returns the combined output and error.
func runCommandVerbose(ctx context.Context, command string, args []string) (string, error) {
	log.GetLogger(ctx).Debugf("🏃 %s (%d) %+v\n", command, len(args), args)

	var cmd *exec.Cmd
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		cmd = _exec.Command("sh", "-c", command)
	} else {
		cmd = _exec.Command(command, args...)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error obtaining stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("error obtaining stderr pipe: %v", err)
	}

	var outBuf, errBuf bytes.Buffer

	scanOutput := func(scanner *bufio.Scanner, prefix string, buf *bytes.Buffer) {
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line + "\n")
			log.GetLogger(ctx).Debugf("%s %s: %s\n", prefix, command, line)
		}
	}

	outScanner := bufio.NewScanner(stdout)
	errScanner := bufio.NewScanner(stderr)

	if err = cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting command: %v", err)
	}

	go scanOutput(outScanner, "stdout", &outBuf)
	go scanOutput(errScanner, "stderr", &errBuf)

	err = cmd.Wait()

	out := outBuf.String() + errBuf.String()
	if err != nil {
		log.GetLogger(ctx).Errorf("\033[31m✗\033[0m %s: %+v\n", command, err)
		return "", fmt.Errorf("%v\n%s", err, clip(out, 500))
	}

	log.GetLogger(ctx).Debugf("🎉 %s: %s\n", command, out)
	return out, nil
}

func execCommand(ctx context.Context, command string, args []string, verbose bool) (string, error) {
	if verbose {
		return runCommandVerbose(ctx, command, args)
	}
	return runCommand(ctx, command, args)
}

func runRestricted(ctx context.Context, vars *api.Vars, command string, args []string) (string, error) {
	if isAllowed(vars.Config.AllowList, command) {
		return execCommand(ctx, command, args, vars.Config.IsVerbose())
	}

	if isDenied(vars.Config.DenyList, command) {
		log.GetLogger(ctx).Errorf("\n\033[31m✗\033[0m restricted\n")
		log.GetLogger(ctx).Infof("%s %v\n\n", command, strings.Join(args, " "))
		if answer, err := bubble.Confirm("Continue?"); err == nil && answer == confirm.Yes {
			return execCommand(ctx, command, args, vars.Config.IsVerbose())
		}

		return "", fmt.Errorf("%s: Not allowed", command)
	}

	// safe, err := evaluateCommand(ctx, vars, command, args)
	// if err != nil {
	// 	return "", err
	// }
	// if safe {
	// 	return execCommand(command, args, vars.Config.Debug)
	// }
	return execCommand(ctx, command, args, vars.Config.IsVerbose())

	// return "", fmt.Errorf("%s %s: Not permitted", command, strings.Join(args, " "))
}
