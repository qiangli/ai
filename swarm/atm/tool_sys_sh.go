package atm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"unicode"

	"github.com/cenkalti/backoff/v4"
	"github.com/pmezard/go-difflib/difflib"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/gitkit"
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
		if envs, err := api.GetMapProp("envs", args); err == nil {
			obj = envs
		} else if envs, err := api.GetArrayProp("envs", args); err == nil {
			if len(envs) > 0 {
				obj = make(map[string]any)
				for _, env := range envs {
					nv := strings.SplitN(env, "=", 2)
					if len(nv) == 2 {
						obj[nv[0]] = nv[1]
					}
				}
			}
		}
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

// SourceEnvs reads a source file containing NAME=VALUE or export NAME=VALUE lines
// and sets valid environment variables. Invalid lines are ignored and warned.
func (r *SystemKit) SourceEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	src, err := api.GetStrProp("source", args)
	if err != nil {
		return nil, err
	}
	if src == "" {
		return nil, fmt.Errorf("missing property: source")
	}
	content, err := api.LoadURIContent(vars.Workspace, src)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(content, "\n")
	obj := make(map[string]any)
	var invalid []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if strings.HasPrefix(l, "#") {
			continue
		}
		// optional export prefix
		if strings.HasPrefix(l, "export ") {
			l = strings.TrimSpace(strings.TrimPrefix(l, "export "))
		}
		// find =
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			invalid = append(invalid, line)
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// strip surrounding quotes from value
		if len(value) >= 2 {
			if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
				value = value[1 : len(value)-1]
			}
		}
		if !isValidEnvName(name) {
			invalid = append(invalid, line)
			continue
		}
		obj[name] = value
	}
	if len(obj) == 0 {
		if len(invalid) > 0 {
			return &api.Result{Value: fmt.Sprintf("No valid environment variables found. Ignored %d invalid lines.", len(invalid))}, nil
		}
		return nil, fmt.Errorf("No environment variables to set.")
	}
	vars.Global.SetEnvs(obj)
	var keys []string
	for k, v := range obj {
		vars.OS.Setenv(k, v)
		keys = append(keys, k)
	}
	msg := fmt.Sprintf("Environment variables %q successfully set.", strings.Join(keys, ","))
	if len(invalid) > 0 {
		msg = msg + " Warning: ignored invalid lines: " + strings.Join(invalid, "; ")
	}
	return &api.Result{Value: msg}, nil
}

func isValidEnvName(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !(unicode.IsLetter(r) || r == '_') {
				return false
			}
		} else {
			if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
				return false
			}
		}
	}
	return true
}

// Diff computes a unified diff between two files using an embedded Go
// diff implementation instead of invoking an external `diff` binary. This
// keeps all functionality inside the application binary as requested.
func (r *SystemKit) Diff(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	aPath, err := api.GetStrProp("a", args)
	if err != nil {
		return "", err
	}
	bPath, err := api.GetStrProp("b", args)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(aPath) == "" || strings.TrimSpace(bPath) == "" {
		return "", fmt.Errorf("both 'a' and 'b' file paths are required")
	}

	// Load file contents via workspace loader to support URIs and virtual FS
	aContent, err := api.LoadURIContent(vars.Workspace, aPath)
	if err != nil {
		return "", fmt.Errorf("failed to load 'a': %w", err)
	}
	bContent, err := api.LoadURIContent(vars.Workspace, bPath)
	if err != nil {
		return "", fmt.Errorf("failed to load 'b': %w", err)
	}

	// Compute unified diff
	// Use difflib's SplitLines helper to prepare line slices.
	d := difflib.UnifiedDiff{
		A:        difflib.SplitLines(aContent),
		B:        difflib.SplitLines(bContent),
		FromFile: aPath,
		ToFile:   bPath,
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(d)
	if err != nil {
		return "", err
	}
	// If there is any diff output, return an error to produce a non-zero
	// exit status (matching the unix `diff` behavior). Include the diff in
	// the error message so callers can see the differences.
	if strings.TrimSpace(text) != "" {
		return text, fmt.Errorf("files differ:\n%s", text)
	}
	// No differences
	return "", nil
}

func (r *SystemKit) Git(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (any, error) {
	var ga gitkit.Args
	ga.ID, _ = api.GetStrProp("id", args)
	ga.User, _ = api.GetStrProp("user", args)
	ga.Payload, _ = api.GetStrProp("payload", args)
	ga.Action, _ = api.GetStrProp("action", args)
	// ga.Dir, _ = api.GetStrProp("dir", args)
	ga.Args, _ = api.GetStrProp("args", args)
	ga.Message, _ = api.GetStrProp("message", args)
	ga.Rev, _ = api.GetStrProp("rev", args)
	ga.Path, _ = api.GetStrProp("path", args)
	// ga.Command, _ = api.GetStrProp("command", args)

	return gitkit.Run(&ga)
}
