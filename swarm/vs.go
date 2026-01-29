package swarm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/qiangli/shell/sh"
	"github.com/qiangli/shell/vfs"
	"github.com/qiangli/shell/vos"
)

type IOE = sh.IOE
type ExecHandler func(context.Context, []string) (bool, error)

type VirtualSystem struct {
	IOE *IOE

	Workspace vfs.Workspace
	System    vos.System

	ExecHandler ExecHandler

	MaxTimeout int
}

func (vs *VirtualSystem) RunScript(ctx context.Context, script string) error {
	r, err := vs.NewRunner(interp.Interactive(true))
	if err != nil {
		return err
	}
	return run(ctx, r, strings.NewReader(script), "")
}

func (vs *VirtualSystem) RunReader(ctx context.Context) error {
	r, err := vs.NewRunner(interp.Interactive(true))
	if err != nil {
		return err
	}
	return run(ctx, r, vs.IOE.Stdin, "")
}

func (vs *VirtualSystem) RunPath(ctx context.Context, path string) error {
	r, err := vs.NewRunner(interp.Interactive(true))
	if err != nil {
		return err
	}
	f, err := vs.Workspace.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return run(ctx, r, f, path)
}

func (vs *VirtualSystem) RunInteractive(ctx context.Context) error {
	r, err := vs.NewRunner(interp.Interactive(true))
	if err != nil {
		return err
	}
	parser := syntax.NewParser()

	fmt.Fprintf(vs.IOE.Stdout, "$ ")
	err = parser.Interactive(vs.IOE.Stdin, func(stmts []*syntax.Stmt) bool {
		if parser.Incomplete() {
			fmt.Fprintf(vs.IOE.Stdout, "> ")
			return true
		}
		// run
		for _, stmt := range stmts {
			err := r.Run(ctx, stmt)
			if err != nil {
				fmt.Fprintf(vs.IOE.Stderr, "error: %s\n", err.Error())
			}
			if r.Exited() {
				vs.System.Exit(0)
				return true
			}
		}
		fmt.Fprintf(vs.IOE.Stdout, "$ ")
		return true
	})
	return err
}

func NewVirtualSystem(ws vfs.Workspace, s vos.System, ioe *IOE) *VirtualSystem {
	return &VirtualSystem{
		Workspace: ws,
		System:    s,
		IOE:       ioe,
	}
}

func NewLocalSystem(roots []string, ioe *IOE) (*VirtualSystem, error) {
	for i, v := range roots {
		abs, err := filepath.Abs(v)
		if err != nil {
			return nil, err
		}
		roots[i] = abs
	}

	fs, err := vfs.NewLocalFS(roots)
	if err != nil {
		return nil, err
	}
	ls, err := vos.NewLocalSystem(fs)
	if err != nil {
		return nil, err
	}

	return NewVirtualSystem(fs, ls, ioe), nil
}

func (vs *VirtualSystem) NewRunner(opts ...interp.RunnerOption) (*interp.Runner, error) {
	r, err := interp.New(opts...)
	if err != nil {
		return nil, err
	}

	interp.OpenHandler(VirtualOpenHandler(vs))(r)
	interp.ReadDirHandler2(VirtualReadDirHandler2(vs))(r)
	interp.StatHandler(VirtualStatHandler(vs))(r)

	//
	interp.CallHandler(VirtualCallHandlerFunc(vs))(r)

	//
	var env = vs.System.Env()
	if len(env) > 0 {
		interp.Env(expand.ListEnviron(env...))(r)
	}

	dir, err := vs.System.Getwd()
	if err != nil {
		return nil, err
	}
	if err := interp.Dir(dir)(r); err != nil {
		return nil, err
	}
	interp.StdIO(vs.IOE.Stdin, vs.IOE.Stdout, vs.IOE.Stderr)(r)

	// exec handlers
	wrap := func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if vs.ExecHandler != nil {
				done, err := vs.ExecHandler(ctx, args)
				if done {
					return err
				}
			}
			return next(ctx, args)
		}
	}
	var middlewares = []func(interp.ExecHandlerFunc) interp.ExecHandlerFunc{
		// custom handler
		wrap,
		// default bash handler
		VirtualExecHandler(vs),
	}
	if err := interp.ExecHandlers(middlewares...)(r); err != nil {
		return nil, err
	}
	return r, nil
}
