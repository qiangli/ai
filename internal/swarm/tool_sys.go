package swarm

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/qiangli/ai/api"
	"github.com/qiangli/ai/internal/swarm/vfs"
	"github.com/qiangli/ai/internal/swarm/vos"
)

var _os vos.System = &vos.VirtualSystem{}
var _exec = _os

var _fs vfs.FileSystem = &vfs.VirtualFS{}

// runCommand executes a shell command with args and returns the output
func runCommand(command string, args []string) (string, error) {
	var out []byte
	var err error
	if len(args) == 0 {
		// LLM sometime sends command and args as a single string
		out, err = _exec.Command("sh", "-c", command).CombinedOutput()
	} else {
		out, err = _exec.Command(command, args...).CombinedOutput()
	}
	if err != nil {
		// send error with out to assist LLM
		return "", fmt.Errorf("%s %v: %v\n %s", command, args, err, out)
	}
	return string(out), nil
}

func runRestricted(ctx context.Context, vars *api.Vars, command string, args []string) (string, error) {
	if isDenied(command) {
		return "", fmt.Errorf("%s: Not allowed", command)
	}
	if isAllowed(command) {
		return runCommand(command, args)
	}

	safe, err := evaluateCommand(ctx, vars, command, args)
	if err != nil {
		return "", err
	}
	if safe {
		return runCommand(command, args)
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
