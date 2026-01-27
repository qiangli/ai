package atm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/cenkalti/backoff/v4"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/resource"
)

var cdNotSupportedError = errors.New(`
*Unsupported Command*: Changing the current working directory isn't supported on the user's system. 
Please use absolute paths for accessing directories and files. 
You may use the fs:list_roots command to identify permissible top-level directories.
`)

// no-op tool that does nothing
func (r *SystemKit) Pass(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return "Success", nil
}

// return an error but does nothing
func (r *SystemKit) Fail(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	msg, _ := api.GetStrProp("report", args)
	return "", fmt.Errorf("%s", msg)
}

// Chdir is not supported
func (r *SystemKit) Cd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return "", cdNotSupportedError
}

func (r *SystemKit) Pwd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.OS.Getwd()
}

func (r *SystemKit) Workspace(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Roots.Workspace.Path, nil
}

func (r *SystemKit) Exec(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	cmd, err := api.GetStrProp("command", args)
	if err != nil {
		return "", err
	}
	if len(cmd) == 0 {
		return "", fmt.Errorf("command is empty")
	}
	// command := argv[0]
	// rest := argv[1:]
	result, err := ExecCommand(ctx, vars.OS, vars, cmd, nil)
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

func (r *SystemKit) Bash(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	// shell handles command/script if empty
	result, err := vars.RootAgent.Shell.Run(ctx, "", args)
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

// Go executes a `go` command (e.g., build/test/vet/list/run) in the user's environment.
//
// This is a minimal wrapper around the shell runner. It does not change directories.
// If you need to run inside a repo, pass an explicit `cd ... && go ...` command.
func (r *SystemKit) Go(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	cmd, err := api.GetStrProp("command", args)
	if err != nil {
		return "", err
	}
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", fmt.Errorf("command is empty")
	}
	// Safety: only allow `go ...` command lines.
	if !strings.HasPrefix(cmd, "go ") && cmd != "go" {
		return "", fmt.Errorf("only 'go ...' commands are allowed")
	}
	result, err := ExecCommand(ctx, vars.OS, vars, cmd, nil)
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

// template is required
func (r *SystemKit) Apply(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	tpl, err := api.GetStrProp("template", args)
	if err != nil {
		return "", err
	}
	if v, err := api.LoadURIContent(vars.Workspace, tpl); err != nil {
		return "", err
	} else {
		tpl = string(v)
	}

	//
	data := BuildEffectiveArgs(vars, nil, args)

	return CheckApplyTemplate(vars.RootAgent.Template, tpl, data)
}

// parse command and copy all vale into args
func (r *SystemKit) Parse(ctx context.Context, vars *api.Vars, name string, args map[string]any) (api.ArgMap, error) {
	result, err := conf.Parse(args["command"])
	if err != nil {
		return nil, err
	}
	// make available in the args???
	maps.Copy(args, result)

	return result, nil
}

// get default template based on format if template is not prvoided.
// tee content to destination specified by output param.
func (r *SystemKit) Format(ctx context.Context, vars *api.Vars, name string, args api.ArgMap) (string, error) {
	var tpl string
	tpl, _ = api.GetStrProp("template", args)
	if tpl != "" {
		if v, err := api.LoadURIContent(vars.Workspace, tpl); err != nil {
			return "", err
		} else {
			tpl = string(v)
		}
	}
	if tpl == "" {
		format, _ := api.GetStrProp("format", args)
		if format == "" {
			format = "markdown"
		}
		tpl = resource.FormatFile(format)
	}

	//
	data := BuildEffectiveArgs(vars, nil, args)

	txt, err := CheckApplyTemplate(vars.RootAgent.Template, tpl, data)
	if err != nil {
		return "", err
	}

	return txt, nil
}

// Run a command and kill it if it runs more than a specified duration
//
// Synopsis:
//
//	timeout [-t duration-string] command [args...]
//
// Description:
//
//	timeout will run the command until it succeeds or too much time has passed.
//	The default timeout is 30s.
//	If no args are given, it will print a usage error.
//
// Example:
//
//	$ timeout echo hi
//	hi
//	$
//	$./timeout -t 5s bash -c 'sleep 40'
//	$ 2022/03/31 14:47:32 signal: killed
//	$./timeout  -t 5s bash -c 'sleep 40'
//	$ 2022/03/31 14:47:40 signal: killed
//	$./timeout  -t 5s bash -c 'sleep 1'
//
// Timeout supports both aciton and command parameters
func (r *SystemKit) Timeout(ctx context.Context, vars *api.Vars, name string, args api.ArgMap) (any, error) {
	var cmdArgs api.ArgMap

	action := args.Action()
	if action == nil {
		// $(command)
		cmdline := args.GetString("command")
		if len(cmdline) == 0 {
			return "", fmt.Errorf("command action is missing")
		}
		nargs, err := conf.Parse(cmdline)
		if err != nil {
			return nil, err
		}
		cmdArgs = nargs
	} else {
		kit, name := api.Kitname(action.Command).Decode()
		args["kit"] = kit
		args["name"] = name
		if kit == "agent" {
			pack, sub := api.Packname(name).Decode()
			args["pack"] = pack
			args["name"] = sub
		}
		cmdArgs = args
	}

	kn := cmdArgs.Kitname()

	duration := args.GetDuration("duration")
	ctx, cancelCtx := context.WithTimeout(ctx, duration)
	defer cancelCtx()

	done := make(chan any)
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
			close(panicChan)
			close(done)
		}()

		// Run the action and handle potential errors.
		result, err := vars.RootAgent.Runner.Run(ctx, kn.ID(), cmdArgs)
		if err != nil {
			panicChan <- err
			return
		}

		done <- result
	}()

	select {
	case p := <-panicChan:
		return nil, p.(error)
	case result := <-done:
		return result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("%q timed out after %v: %v", kn, duration, ctx.Err())
	}
}

// Run a command, repeatedly, until it succeeds or we are out of time
//
// Synopsis:
//
//	backoff -v [-t duration-string] command [args...]
//
// Description:
//
//	backoff will run the command until it succeeds or a timeout has passed.
//	The default timeout is 30s.
//	If -v is set, it will show what it is running, each time it is tried.
//	If no args are given, it will print command help.
//
// Example:
//
//	$ backoff echo hi
//	hi
//	$
//	$ backoff -v -t=2s false
//	  2022/03/31 14:29:37 Run ["false"]
//	  2022/03/31 14:29:37 Set timeout to 2s
//	  2022/03/31 14:29:37 "false" []:exit status 1
//	  2022/03/31 14:29:38 "false" []:exit status 1
//	  2022/03/31 14:29:39 "false" []:exit status 1
//	  2022/03/31 14:29:39 Error: exit status 1
func (r *SystemKit) Backoff(ctx context.Context, vars *api.Vars, name string, args api.ArgMap) (any, error) {
	var cmdArgs api.ArgMap

	action := args.Action()
	if action == nil {
		// $(command)
		cmdline := args.GetString("command")
		if len(cmdline) == 0 {
			return "", fmt.Errorf("command action is missing")
		}
		nargs, err := conf.Parse(cmdline)
		if err != nil {
			return nil, err
		}
		cmdArgs = nargs
	} else {
		kit, name := api.Kitname(action.Command).Decode()
		args["kit"] = kit
		args["name"] = name
		if kit == "agent" {
			pack, sub := api.Packname(name).Decode()
			args["pack"] = pack
			args["name"] = sub
		}
		cmdArgs = args
	}

	kn := cmdArgs.Kitname()

	duration := args.GetDuration("duration")

	var result any

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = duration
	f := func() error {
		v, err := vars.RootAgent.Runner.Run(ctx, kn.ID(), cmdArgs)
		result = v
		return err
	}

	if err := backoff.Retry(f, b); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *SystemKit) GetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	keys, err := api.GetArrayProp("keys", args)
	if err != nil {
		return nil, err
	}

	envs := vars.Global.GetEnvs(keys)
	b, err := json.Marshal(envs)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: string(b),
	}, nil
}

// Export object set by key "envs" or all args if key is not found.
func (r *SystemKit) SetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	// TODO merge to make a single source of truth
	if len(args) == 0 {
		return nil, fmt.Errorf("Error: Expected environment variables to set but received none.")
	}
	var obj map[string]any
	if _, ok := args["envs"]; ok {
		v, err := api.GetMapProp("envs", args)
		if err != nil {
			return nil, err
		}
		obj = v
	} else {
		// set all
		obj = args
	}
	if len(obj) == 0 {
		return nil, fmt.Errorf("No environment variables to set.")
	}
	vars.Global.SetEnvs(obj)
	var keys []string
	for k, v := range obj {
		vars.OS.Setenv(k, v)
		keys = append(keys, k)
	}
	return &api.Result{
		Value: fmt.Sprintf("Environment variables %q successfully set.", strings.Join(keys, ",")),
	}, nil
}

func (r *SystemKit) UnsetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	keys, err := api.GetArrayProp("keys", args)
	if err != nil {
		return nil, err
	}

	vars.Global.UnsetEnvs(keys)
	// TODO delete env from OS
	for _, k := range keys {
		vars.OS.Setenv(k, "")
	}
	return &api.Result{
		Value: fmt.Sprintf("Environment variables %q successfully cleared.", strings.Join(keys, ",")),
	}, nil
}
