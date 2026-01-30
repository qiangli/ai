package swarm

import (
	"context"
	"fmt"
	"os"
	// "path/filepath"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/qiangli/shell/sh"
	// "github.com/qiangli/shell/vfs"
	// "github.com/qiangli/shell/vos"
	"github.com/qiangli/ai/swarm/api"
)

type IOE = sh.IOE
type ExecHandler func(context.Context, []string) (bool, error)

type VirtualSystem struct {
	ioe *IOE

	// fs vfs.Workspace
	// os    vos.System

	// ExecHandler ExecHandler
	vars  *api.Vars
	agent *api.Agent

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
	return run(ctx, r, vs.ioe.Stdin, "")
}

func (vs *VirtualSystem) RunPath(ctx context.Context, path string) error {
	r, err := vs.NewRunner(interp.Interactive(true))
	if err != nil {
		return err
	}
	f, err := vs.vars.Workspace.OpenFile(path, os.O_RDONLY, 0)
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

	fmt.Fprintf(vs.ioe.Stdout, "$ ")
	err = parser.Interactive(vs.ioe.Stdin, func(stmts []*syntax.Stmt) bool {
		if parser.Incomplete() {
			fmt.Fprintf(vs.ioe.Stdout, "> ")
			return true
		}
		// run
		for _, stmt := range stmts {
			err := r.Run(ctx, stmt)
			if err != nil {
				fmt.Fprintf(vs.ioe.Stderr, "error: %s\n", err.Error())
			}
			if r.Exited() {
				vs.vars.OS.Exit(0)
				return true
			}
		}
		fmt.Fprintf(vs.ioe.Stdout, "$ ")
		return true
	})
	return err
}

func NewVirtualSystem(vars *api.Vars, agent *api.Agent, ioe *IOE) *VirtualSystem {
	return &VirtualSystem{
		vars:  vars,
		agent: agent,
		ioe:   ioe,
	}
}

// func NewLocalSystem(agent *api.Agent, roots []string, ioe *IOE) (*VirtualSystem, error) {
// 	for i, v := range roots {
// 		abs, err := filepath.Abs(v)
// 		if err != nil {
// 			return nil, err
// 		}
// 		roots[i] = abs
// 	}

// 	fs, err := vfs.NewLocalFS(roots)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ls, err := vos.NewLocalSystem(fs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return NewVirtualSystem(fs, ls, agent, ioe), nil
// }

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
	var env = vs.vars.OS.Env()
	if len(env) > 0 {
		interp.Env(expand.ListEnviron(env...))(r)
	}

	dir, err := vs.vars.OS.Getwd()
	if err != nil {
		return nil, err
	}
	if err := interp.Dir(dir)(r); err != nil {
		return nil, err
	}
	interp.StdIO(vs.ioe.Stdin, vs.ioe.Stdout, vs.ioe.Stderr)(r)

	var middlewares = []func(interp.ExecHandlerFunc) interp.ExecHandlerFunc{
		VirtualExecHandler(vs),
	}
	if err := interp.ExecHandlers(middlewares...)(r); err != nil {
		return nil, err
	}
	return r, nil
}
