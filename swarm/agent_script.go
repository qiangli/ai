package swarm

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"slices"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
	// "github.com/qiangli/shell/sh"
)

type AgentScriptRunner struct {
	vars  *api.Vars
	agent *api.Agent
}

func NewAgentScriptRunner(vars *api.Vars, agent *api.Agent) api.ActionRunner {
	return &AgentScriptRunner{
		vars:  vars,
		agent: agent,
	}
}

func (r *AgentScriptRunner) loadScript(v string) (string, error) {
	return api.LoadURIContent(r.vars.Workspace, v)
}

// Run command or script.
// If script is empty, read command or script from args.
func (r *AgentScriptRunner) Run(ctx context.Context, script string, args map[string]any) (any, error) {
	if script == "" && args != nil {
		if c, ok := args["command"]; ok {
			script = api.ToString(c)
		} else {
			s, ok := args["script"]
			if !ok {
				return nil, fmt.Errorf("script not found")
			}
			if v, err := r.loadScript(api.ToString(s)); err == nil {
				script = v
			}
		}
	}

	if script == "" {
		return nil, fmt.Errorf("missing bash command/script")
	}

	// bash script
	var b bytes.Buffer
	ioe := &IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
	vs := NewVirtualSystem(r.vars.Workspace, r.vars.OS, ioe)

	// pass current env
	// required to run commands: /sh:go
	for _, env := range os.Environ() {
		if env != "" {
			nv := strings.SplitN(env, "=", 2)
			if len(nv) == 2 {
				vs.System.Setenv(nv[0], nv[1])
			}
		}
	}
	// set global env for bash script
	// TODO batch set
	for k, v := range r.vars.Global.GetAllEnvs() {
		vs.System.Setenv(k, v)
	}
	// make args available as env
	for k, v := range args {
		vs.System.Setenv(k, v)
	}

	vs.ExecHandler = r.newExecHandler(vs, args)

	// run bash interpreter
	// and make the error/reslt available in args
	err := vs.RunScript(ctx, script)

	// FIXME: translate error into exit status and respect set -e
	if err != nil {
		if exit := vs.System.Getenv("option_exit"); exit == "true" {
			vs.System.Exit(1)
		}
		return nil, err
	}
	vs.System.Exit(0)
	result := &api.Result{
		Value: b.String(),
	}
	return result, nil
}

// type ExecHandlerFunc func(ctx context.Context, args []string) error
// type CallHandlerFunc func(ctx context.Context, args []string) ([]string, error)

func (r *AgentScriptRunner) newExecHandler(vs *VirtualSystem, _ map[string]any) ExecHandler {
	return func(ctx context.Context, args []string) (bool, error) {
		if r.agent == nil {
			return true, fmt.Errorf("script: missing agent")
		}
		log.GetLogger(ctx).Debugf("script agent: %s args: %+v\n", r.agent.Name, args)

		cmd := strings.ToLower(args[0])
		if conf.IsAction(cmd) {
			kit, name := api.Kitname(cmd).Decode()
			if kit != "" && name != "" {
				at, err := conf.ParseActionArgs(args)
				if err != nil {
					return true, err
				}

				in := atm.BuildEffectiveArgs(r.vars, r.agent, at)

				_, err = r.run(ctx, vs, in)
				if err != nil {
					return true, err
				}

				return true, nil
			}
			// system command - continue
		}

		// internal list
		allowed := []string{"env", "printenv"}
		if slices.Contains(allowed, args[0]) {
			out, err := doBashCustom(r.vars, vs, args)
			fmt.Fprintf(vs.IOE.Stdout, "%v", out)
			if err != nil {
				fmt.Fprintln(vs.IOE.Stderr, err.Error())
			}
			return true, err
		}

		// bash core utils
		if did, err := RunCoreUtils(ctx, vs, args); did {
			// TDDO core util output?
			return did, err
		}

		// TODO restricted
		// block other commands
		out, err := atm.ExecCommand(ctx, r.vars.OS, r.vars, args[0], args[1:])

		// out already has stdout/stder combined
		fmt.Fprintf(vs.IOE.Stdout, "%v", out)
		return true, err
	}
}

func (r *AgentScriptRunner) run(ctx context.Context, vs *VirtualSystem, args api.ArgMap) (*api.Result, error) {
	result, err := api.Exec(ctx, r.agent.Runner, args)
	if result != nil {
		fmt.Fprintln(vs.IOE.Stdout, result.Value)
	}
	if err != nil {
		fmt.Fprintln(vs.IOE.Stderr, err.Error())
		// check set -e
		vs.System.Exit(1)
		return nil, err
	}
	return result, nil
}

func doBashCustom(vars *api.Vars, vs *VirtualSystem, args []string) (string, error) {
	printenv := func() string {
		var envs []string
		for k, v := range vs.System.Environ() {
			envs = append(envs, fmt.Sprintf("%s=%v", k, v))
		}
		return strings.Join(envs, "\n") + "\n"
	}
	setenv := func(key, val string) {
		vars.Global.Set(key, val)
		vs.System.Setenv(key, val)
	}
	unsetenv := func(key string) {
		vars.Global.Unset(key)
		// TODO add unset
		vs.System.Setenv(key, "")
	}

	switch args[0] {
	case "printenv":
		return printenv(), nil
	case "env":
		if len(args) == 1 {
			return printenv(), nil
		}
		// setenv
		for _, kv := range args[1:] {
			k, v := split2(kv, "=", "")
			setenv(k, v)
		}
	case "setenv":
		for _, kv := range args[1:] {
			k, v := split2(kv, "=", "")
			setenv(k, v)
		}
	case "unsetenv":
		for _, k := range args[1:] {
			unsetenv(k)
		}
	default:
	}
	return "", nil
}
