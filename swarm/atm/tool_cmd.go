package atm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/vos"
)

// RunCommand executes a shell command with args and returns the output
func RunCommand(ctx context.Context, vs vos.System, command string, args []string) (string, error) {
	log.GetLogger(ctx).Debugf("🏃 %s (%d) %+v\n", command, len(args), args)

	var out []byte
	var err error
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		out, err = vs.Command("sh", "-c", command).CombinedOutput()
	} else {
		out, err = vs.Command(command, args...).CombinedOutput()
	}
	if err != nil {
		log.GetLogger(ctx).Errorf("❌ %s: %+v\n", command, err)
		return "", fmt.Errorf("%v\n%s", err, clip(string(out), 500))
	}

	log.GetLogger(ctx).Debugf("🎉 %s: %s\n", command, out)
	return string(out), nil
}

// RunCommandVerbose executes a shell command with arguments,
// prints stdout/stderr in real-time, and returns the combined output and error.
func RunCommandVerbose(ctx context.Context, vs vos.System, command string, args []string) (string, error) {
	log.GetLogger(ctx).Debugf("🏃 %s (%d) %+v\n", command, len(args), args)

	var cmd *exec.Cmd
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		cmd = vs.Command("sh", "-c", command)
	} else {
		cmd = vs.Command(command, args...)
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
		log.GetLogger(ctx).Errorf("❌ %s: %+v\n", command, err)
		return "", fmt.Errorf("%v\n%s", err, clip(out, 500))
	}

	log.GetLogger(ctx).Debugf("🎉 %s: %s\n", command, out)
	return out, nil
}

func ExecCommand(ctx context.Context, vs vos.System, vars *api.Vars, command string, args []string) (string, error) {
	return RunCommand(ctx, vs, command, args)
}
