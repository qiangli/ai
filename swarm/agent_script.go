package swarm

import (
	"bytes"
	"context"
	"fmt"

	"slices"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
)

type AgentScriptRunner struct {
	// sw    *Swarm
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
	return api.LoadURIContent(r.vars.RTE.Workspace, v)
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
	ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
	vs := sh.NewVirtualSystem(r.vars.RTE.OS, r.vars.RTE.Workspace, ioe)

	// set global env for bash script
	// TODO batch set
	for k, v := range r.vars.Global.GetAllEnvs() {
		vs.System.Setenv(k, v)
	}

	vs.ExecHandler = r.newExecHandler(vs, args)

	// run bash interpreter
	// and make the error/reslt available in args
	err := vs.RunScript(ctx, script)

	if err != nil {
		// args["error"] = err.Error()
		return nil, err
	}
	// copy back env
	r.vars.Global.AddEnvs(vs.System.Environ())

	result := &api.Result{
		Value: b.String(),
	}
	// args["result"] = result
	return result, nil
}

func (r *AgentScriptRunner) newExecHandler(vs *sh.VirtualSystem, _ map[string]any) sh.ExecHandler {
	return func(ctx context.Context, args []string) (bool, error) {
		if r.agent == nil {
			return true, fmt.Errorf("script: missing agent")
		}
		log.GetLogger(ctx).Debugf("script agent: %s args: %+v\n", r.agent.Name, args)

		if conf.IsAction(strings.ToLower(args[0])) {
			// log.GetLogger(ctx).Debugf("script: ai agent/tool: %+v\n", args)

			at, err := conf.ParseActionArgs(args)
			if err != nil {
				return true, err
			}

			// // consistant with template
			// var in = make(map[string]any)
			// // defaults
			// maps.Copy(in, r.agent.Arguments)
			// //
			// maps.Copy(in, r.vars.Global.GetAllEnvs())
			// // share or not share?
			// // maps.Copy(in, envs)
			// maps.Copy(in, at)

			in := atm.BuildEffectiveArgs(r.vars, r.agent, at)

			_, err = r.run(ctx, vs, in)
			if err != nil {
				return true, err
			}

			return true, nil
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
		if did, err := sh.RunCoreUtils(ctx, vs, args); did {
			// TDDO core util output?
			return did, err
		}

		// TODO restricted
		// block other commands
		out, err := atm.ExecCommand(ctx, r.vars.RTE.OS, r.vars, args[0], args[1:])
		// out already has stdout/stder combined
		fmt.Fprintf(vs.IOE.Stdout, "%v", out)
		return true, err
	}
}

func (r *AgentScriptRunner) run(ctx context.Context, vs *sh.VirtualSystem, args api.ArgMap) (*api.Result, error) {
	result, err := api.Exec(ctx, r.agent.Runner, args)
	if result != nil {
		fmt.Fprintln(vs.IOE.Stdout, result.Value)
	}
	if err != nil {
		fmt.Fprintln(vs.IOE.Stderr, err.Error())
		return nil, err
	}
	return result, nil
}

func doBashCustom(vars *api.Vars, vs *sh.VirtualSystem, args []string) (string, error) {
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
