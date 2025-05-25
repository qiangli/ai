package swarm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/qiangli/ai/bubble"
	"github.com/qiangli/ai/bubble/confirm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/vfs"
	"github.com/qiangli/ai/swarm/vos"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os

var _fs vfs.FileSystem = &vfs.VirtualFS{}

// runCommand executes a shell command with args and returns the output
func runCommand(command string, args []string) (string, error) {
	log.Debugf("🏃 %s (%d) %+v\n", command, len(args), args)

	var out []byte
	var err error
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		out, err = _exec.Command("sh", "-c", command).CombinedOutput()
	} else {
		out, err = _exec.Command(command, args...).CombinedOutput()
	}
	if err != nil {
		log.Errorf("\033[31m✗\033[0m %s: %+v\n", command, err)
		return "", fmt.Errorf("%v\n%s", err, clip(string(out), 500))
	}

	log.Debugf("🎉 %s: %s\n", command, out)
	return string(out), nil
}

// runCommandVerbose executes a shell command with arguments,
// prints stdout/stderr in real-time, and returns the combined output and error.
func runCommandVerbose(command string, args []string) (string, error) {
	log.Debugf("🏃 %s (%d) %+v\n", command, len(args), args)

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
			log.Debugf("%s %s: %s\n", prefix, command, line)
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
		log.Errorf("\033[31m✗\033[0m %s: %+v\n", command, err)
		return "", fmt.Errorf("%v\n%s", err, clip(out, 500))
	}

	log.Debugf("🎉 %s: %s\n", command, out)
	return out, nil
}

func execCommand(command string, args []string, verbose bool) (string, error) {
	if verbose {
		return runCommandVerbose(command, args)
	}
	return runCommand(command, args)
}

func runRestricted(ctx context.Context, vars *api.Vars, command string, args []string) (string, error) {
	if isAllowed(vars.Config.AllowList, command) {
		return execCommand(command, args, vars.Config.Debug)
	}

	if isDenied(vars.Config.DenyList, command) {
		log.Errorf("\n\033[31m✗\033[0m restricted\n")
		log.Infof("%s %v\n\n", command, strings.Join(args, " "))
		if answer, err := bubble.Confirm("Continue?"); err == nil && answer == confirm.Yes {
			return execCommand(command, args, vars.Config.Debug)
		}

		return "", fmt.Errorf("%s: Not allowed", command)
	}

	safe, err := evaluateCommand(ctx, vars, command, args)
	if err != nil {
		return "", err
	}
	if safe {
		return execCommand(command, args, vars.Config.Debug)
	}

	return "", fmt.Errorf("%s %s: Not permitted", command, strings.Join(args, " "))
}

// if required properties is not missing and is an array of strings
// check if the required properties are present
func isRequired(key string, props map[string]any) bool {
	val, ok := props["required"]
	if !ok {
		return false
	}
	items, ok := val.([]string)
	if !ok {
		return false
	}
	for _, v := range items {
		if v == key {
			return true
		}
	}
	return false
}

func GetStrProp(key string, props map[string]any) (string, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return "", fmt.Errorf("missing property: %s", key)
		}
		return "", nil
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("property '%s' must be a string", key)
	}
	return str, nil
}

func GetIntProp(key string, props map[string]any) (int, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return 0, fmt.Errorf("missing property: %s", key)
		}
		return 0, nil
	}
	switch v := val.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		s := fmt.Sprintf("%v", val)
		return strconv.Atoi(s)
	}
}

func GetArrayProp(key string, props map[string]any) ([]string, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return nil, fmt.Errorf("missing property: %s", key)
		}
		return []string{}, nil
	}
	items, ok := val.([]any)
	if ok {
		strs := make([]string, len(items))
		for i, v := range items {
			str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be an array of strings", key)
			}
			strs[i] = str
		}
		return strs, nil
	}

	strs, ok := val.([]string)
	if !ok {
		if isRequired(key, props) {
			return nil, fmt.Errorf("%s must be an array of strings", key)
		}
		return []string{}, nil
	}
	return strs, nil
}
