package swarm

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"

	"github.com/qiangli/shell/vfs"
)

func VirtualOpenHandler(vs *VirtualSystem) interp.OpenHandlerFunc {
	return func(ctx context.Context, path string, flag int, perm fs.FileMode) (io.ReadWriteCloser, error) {
		hc := interp.HandlerCtx(ctx)

		//
		if runtime.GOOS == "windows" && path == "/dev/null" {
			path = "NUL"
			// Work around https://go.dev/issue/71752, where Go 1.24 started giving
			// "Invalid handle" errors when opening "NUL" with O_TRUNC.
			// TODO: hopefully remove this in the future once the bug is fixed.
			flag &^= os.O_TRUNC
		} else if path != "" && !filepath.IsAbs(path) {
			path = filepath.Join(hc.Dir, path)
		}
		return vs.vars.Workspace.OpenFile(path, flag, perm)
	}
}

func VirtualReadDirHandler2(vs *VirtualSystem) interp.ReadDirHandlerFunc2 {
	return func(ctx context.Context, path string) ([]fs.DirEntry, error) {
		return vs.vars.Workspace.ReadDir(path)
	}
}

func VirtualStatHandler(vs *VirtualSystem) interp.StatHandlerFunc {
	return func(ctx context.Context, path string, followSymlinks bool) (fs.FileInfo, error) {
		if v, ok := vs.vars.Workspace.(vfs.FileStat); ok {
			if !followSymlinks {
				return v.Lstat(path)
			} else {
				return v.Stat(path)
			}
		}
		if followSymlinks {
			return nil, fmt.Errorf("not supported")
		}
		return vs.vars.Workspace.GetFileInfo(path)
	}
}

func execEnv(env expand.Environ) []string {
	list := make([]string, 0, 64)
	for name, vr := range env.Each {
		if !vr.IsSet() {
			// If a variable is set globally but unset in the
			// runner, we need to ensure it's not part of the final
			// list. Seems like zeroing the element is enough.
			// This is a linear search, but this scenario should be
			// rare, and the number of variables shouldn't be large.
			for i, kv := range list {
				if strings.HasPrefix(kv, name+"=") {
					list[i] = ""
				}
			}
		}
		if vr.Exported && vr.Kind == expand.String {
			list = append(list, name+"="+vr.String())
		}
	}
	return list
}

func VirtualExecHandler(vs *VirtualSystem) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	handle := func(ctx context.Context, args []string) error {
		hc := interp.HandlerCtx(ctx)

		err := HandleAction(ctx, vs, args)

		switch err := err.(type) {
		case *exec.ExitError:
			// Windows and Plan9 do not have support for [syscall.WaitStatus]
			// with methods like Signaled and Signal, so for those, [waitStatus] is a no-op.
			// Note: [waitStatus] is an alias [syscall.WaitStatus]
			// if status, ok := err.Sys().(waitStatus); ok && status.Signaled() {
			// 	if ctx.Err() != nil {
			// 		return ctx.Err()
			// 	}
			// 	return interp.ExitStatus(128 + status.Signal())
			// }
			return interp.ExitStatus(err.ExitCode())
		case *exec.Error:
			// did not start
			fmt.Fprintf(hc.Stderr, "%v\n", err)
			return interp.ExitStatus(127)
		default:
			return err
		}
	}

	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			return handle(ctx, args)
		}
	}
}

// TODO
// set:
// -e  Exit immediately if a command exits with a non-zero status.
// -o  option-name
//
//				pipefail   the return value of a pipeline is the status of
//	                       the last command to exit with a non-zero status,
//	                       or zero if no command exited with a non-zero status
//
// -u  Treat unset variables as an error when substituting.
// -x  Print commands and their arguments as they are executed.
func VirtualCallHandlerFunc(vs *VirtualSystem) interp.CallHandlerFunc {
	return func(ctx context.Context, args []string) ([]string, error) {
		switch args[0] {
		case "cd":
			return nil, fmt.Errorf("Changing the current working directory is not supported\nFor legacy bash scripts relying on `cd`, use the 'sh:exec' tool, e.g., sh:exec --command '/bin/bash </script/file>'\n")
		case "exec":
			return nil, fmt.Errorf("System exec command not supported: %v\nUse the 'sh:exec' tool, e.g., sh:exec --command '...'\n ", args)
		case "set":
			// parse -e | -o pipefail and set as env: option_exit = true|false option_pipefail = true | false
			for i, arg := range args[1:] {
				var opt bool
				if strings.HasPrefix(arg, "-") {
					// Handle combined options like -xeuo
					for _, v := range arg[1:] {
						switch string(v) {
						case "e":
							vs.vars.OS.Setenv("option_exit", "true")
						case "o":
							opt = true
						}
					}
				}
				if (opt || arg == "-o") && i+1 < len(args[1:]) {
					// Handle -o pipefail or other options
					if args[i+2] == "pipefail" {
						vs.vars.OS.Setenv("option_pipefail", "true")
					}
				}
			}
		default:
		}
		return args, nil
	}
}

// return true if the last elemment is or ends in sh/bash
func IsShell(s string) bool {
	if slices.Contains([]string{"bash", "sh"}, path.Base(s)) {
		return true
	}
	if slices.Contains([]string{".bash", ".sh"}, path.Base(path.Ext(s))) {
		return true
	}
	return false
}

// deny list
func IsRestricted(s string) bool {
	return false
}

func run(ctx context.Context, r *interp.Runner, reader io.Reader, name string) error {
	prog, err := syntax.NewParser().Parse(reader, name)
	if err != nil {
		return err
	}
	r.Reset()
	return r.Run(ctx, prog)
}

// runCommandexecutes a command with a context-based timeout and handles termination.
func runCommand(ctx context.Context, vs *VirtualSystem, args []string) error {
	hc := interp.HandlerCtx(ctx)

	var maxTime = 15 * time.Minute
	if vs.MaxTimeout > 0 {
		maxTime = time.Duration(vs.MaxTimeout)
	}

	path, err := interp.LookPathDir(hc.Dir, hc.Env, args[0])
	if err != nil {
		fmt.Fprintln(hc.Stderr, err)
		return interp.ExitStatus(127)
	}

	// cmd := vs.System.Command(args[0], args[1:]...)
	cmd := vs.vars.OS.Command(path)
	cmd.Path = path
	cmd.Args = args
	cmd.Env = execEnv(hc.Env)
	cmd.Dir = hc.Dir
	cmd.Stdin = hc.Stdin
	cmd.Stdout = hc.Stdout
	cmd.Stderr = hc.Stderr

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Create a context with timeout if maxTime is specified
	var cancel context.CancelFunc
	if maxTime > 0 {
		ctx, cancel = context.WithTimeout(ctx, maxTime)
		defer cancel()
	}

	// Handle command termination on context cancellation
	stopf := context.AfterFunc(ctx, func() {
		if runtime.GOOS == "windows" {
			// On Windows, directly kill the process as signals like SIGINT are not supported
			_ = cmd.Process.Kill()
			return
		}
		// Send interrupt signal for graceful shutdown
		_ = cmd.Process.Signal(os.Interrupt)
		// Give a very short grace period for the process to terminate gracefully
		graceTimer := time.NewTimer(100 * time.Millisecond)
		defer graceTimer.Stop()
		select {
		case <-graceTimer.C:
			// If the process doesn't stop within the grace period, forcefully kill it
			_ = cmd.Process.Kill()
		case <-ctx.Done():
			// If context is cancelled again or already done, do nothing extra
		}
	})
	defer stopf()

	return cmd.Wait()
}
