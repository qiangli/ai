package lang

import (
	"bytes"
	"context"
	"io/fs"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Golang interepter
func Golang(ctx context.Context, f fs.FS, env []string, script string, args []string) (any, error) {
	var b bytes.Buffer
	i := interp.New(interp.Options{
		Stdout:               &b,
		Stderr:               &b,
		Env:                  env,
		Args:                 args,
		SourcecodeFilesystem: f,
		Unrestricted:         false,
	})
	i.Use(stdlib.Symbols)
	_, err := i.Eval(script)
	if err != nil {
		return nil, err
	}
	return b.String(), nil
}
