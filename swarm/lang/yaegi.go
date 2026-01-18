package lang

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"github.com/qiangli/ai/swarm/api"
)

// Golang interepter
func Golang(ctx context.Context, f fs.FS, global map[string]any, script string, input map[string]any) (any, error) {
	var b bytes.Buffer
	goPath := os.Getenv("GOPATH")
	env := mergetEnvArgs(global, input)
	args := toStringArgs(input)
	i := interp.New(interp.Options{
		Stdout:               &b,
		Stderr:               &b,
		Env:                  env,
		Args:                 args,
		SourcecodeFilesystem: f,
		Unrestricted:         false,
		GoPath:               goPath,
	})
	i.Use(stdlib.Symbols)
	_, err := i.Eval(script)
	if err != nil {
		return nil, err
	}
	return b.String(), nil
}

// convert args map to string args suitable for command line
// name=value -> --name "value"
func toStringArgs(args map[string]any) []string {
	var strArgs []string

	for name, value := range args {
		strArgs = append(strArgs, fmt.Sprintf("--%s", name))
		strArgs = append(strArgs, fmt.Sprintf("%v", value))
	}

	return strArgs
}

// merge env and args with args having a higher precedence
func mergetEnvArgs(env map[string]any, args map[string]any) []string {
	var all = make(map[string]string)

	for k, v := range env {
		all[k] = api.ToString(v)
	}
	for k, v := range args {
		all[k] = api.ToString(v)
	}

	var result []string
	for k, v := range all {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}

	return result
}
