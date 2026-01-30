package swarm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"mvdan.cc/sh/v3/interp"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
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
	vs := NewVirtualSystem(r.vars, r.agent, ioe)

	// pass current env
	// required to run commands: /sh:go
	for _, env := range os.Environ() {
		if env != "" {
			nv := strings.SplitN(env, "=", 2)
			if len(nv) == 2 {
				vs.vars.OS.Setenv(nv[0], nv[1])
			}
		}
	}
	// set global env for bash script
	// TODO batch set
	for k, v := range r.vars.Global.GetAllEnvs() {
		vs.vars.OS.Setenv(k, v)
	}
	// make args available as env
	for k, v := range args {
		vs.vars.OS.Setenv(k, v)
	}

	// vs.ExecHandler = newExecHandler(r.vars, r.agent, vs, args)

	// run bash interpreter
	// and make the error/reslt available in args
	err := vs.RunScript(ctx, script)

	// FIXME: translate error into exit status and respect set -e
	if err != nil {
		if exit := vs.vars.OS.Getenv("option_exit"); exit == "true" {
			vs.vars.OS.Exit(1)
		}
		return nil, err
	}
	vs.vars.OS.Exit(0)
	result := &api.Result{
		Value: b.String(),
	}
	return result, nil
}

func HandleAction(ctx context.Context, vs *VirtualSystem, args []string) error {
	hc := interp.HandlerCtx(ctx)

	// agent is always required
	if vs.agent == nil {
		return fmt.Errorf("script: missing agent")
	}
	log.GetLogger(ctx).Debugf("script agent: %s args: %+v\n", vs.agent.Name, args)

	cmd := strings.ToLower(args[0])

	// ai extended feature; agent/tool support
	if conf.IsAction(cmd) {
		kit, name := api.Kitname(cmd).Decode()
		if kit != "" && name != "" {
			at, err := conf.ParseActionArgs(args)
			if err != nil {
				return err
			}
			in := atm.BuildEffectiveArgs(vs.vars, vs.agent, at)
			result, err := api.Exec(ctx, vs.agent.Runner, in)
			if result != nil {
				fmt.Fprintln(hc.Stdout, result.Value)
			}
			if err != nil {
				fmt.Fprintln(hc.Stderr, err.Error())
				return err
			}
			return nil
		}

		// kit == ""
		// system command - continue
	}

	// exec
	if cmd == "exec" {
		fmt.Fprintf(hc.Stderr, "System exec command not supported: %v\nUse tool 'sh:exec'", args)
	}

	// allow bash builtin
	if interp.IsBuiltin(cmd) {
		return hc.Builtin(ctx, args)
	}

	// custom util
	// TODO merge with core tuil
	allowed := []string{"env", "printenv"}
	if slices.Contains(allowed, cmd) {
		out, err := runBashCustom(vs, args)
		fmt.Fprintf(hc.Stdout, "%v", out)
		if err != nil {
			fmt.Fprintln(hc.Stderr, err.Error())
		}
		return err
	}

	// bash core utils
	if IsCoreUtils(cmd) {
		err := RunCoreUtil(ctx, vs, args)
		return err
	}

	// bash subshell
	if IsShell(cmd) {
		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err
		}
		in := atm.BuildEffectiveArgs(vs.vars, vs.agent, at)
		subsh := NewAgentScriptRunner(vs.vars, vs.agent)
		out, err := subsh.Run(ctx, cmd, in)
		result := api.ToResult(out)
		if result != nil {
			fmt.Fprintln(hc.Stdout, result.Value)
		}
		if err != nil {
			fmt.Fprintln(hc.Stderr, err.Error())
			return err
		}
		return nil
	}

	// TODO restricted
	if IsRestricted(cmd) {
		fmt.Fprintf(hc.Stderr, "command not supported: %s %+v\n", cmd, args[1:])
		return nil
	}

	// command
	if err := runCommandWithTimeout(ctx, vs, args); err != nil {
		return err
	}

	return nil
}

func runBashCustom(vs *VirtualSystem, args []string) (string, error) {
	printenv := func() string {
		var envs []string
		for k, v := range vs.vars.OS.Environ() {
			envs = append(envs, fmt.Sprintf("%s=%v", k, v))
		}
		return strings.Join(envs, "\n") + "\n"
	}
	setenv := func(key, val string) {
		vs.vars.Global.Set(key, val)
		vs.vars.OS.Setenv(key, val)
	}
	unsetenv := func(key string) {
		vs.vars.Global.Unset(key)
		// TODO add unset
		vs.vars.OS.Setenv(key, "")
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
