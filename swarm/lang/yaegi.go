package lang

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Golang interepter
func Golang(ctx context.Context, f fs.FS, global []string, script string, args []string) (any, error) {
	var b bytes.Buffer
	goPath := os.Getenv("GOPATH")
	env := mergetEnvArg(global, args)
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

// merge env and args with args having a higher precedence
func mergetEnvArg(env []string, args []string) []string {
	var all = make(map[string]string)

	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		key := parts[0]
		if len(parts) == 2 {
			all[key] = parts[1]
		}
	}

	for _, a := range args {
		parts := strings.SplitN(a, "=", 2)
		key := parts[0]
		if len(parts) == 2 {
			all[key] = parts[1]
		}
	}

	var result []string
	for key, value := range all {
		result = append(result, key+"="+value)
	}

	return result
}
