package lang

import (
	"bytes"
	"context"
	"io/fs"
	"os"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

// Golang interepter
func Golang(ctx context.Context, f fs.FS, env []string, script string, args []string) (any, error) {
	var b bytes.Buffer
	goPath := os.Getenv("GOPATH")
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
