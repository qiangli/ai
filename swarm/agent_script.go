package swarm

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
)

type AgentScriptRunner struct {
	sw     *Swarm
	parent *api.Agent
}

func NewAgentScriptRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	return &AgentScriptRunner{
		sw:     sw,
		parent: agent,
	}
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
			if v, err := r.sw.LoadScript(api.ToString(s)); err == nil {
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
	vs := sh.NewVirtualSystem(r.sw.OS, r.sw.Workspace, ioe)

	// set global env for bash script
	// TODO batch set
	for k, v := range r.sw.Vars.Global.GetAllEnvs() {
		vs.System.Setenv(k, v)
	}
	// for k, v := range r.parent.Environment.GetAllEnvs() {
	// 	vs.System.Setenv(k, v)
	// }

	vs.ExecHandler = r.newExecHandler(vs, args)

	// run bash interpreter
	// and make the error/reslt available in args
	err := vs.RunScript(ctx, script)

	if err != nil {
		args["error"] = err.Error()
		return nil, err
	}
	// copy back env
	r.sw.Vars.Global.AddEnvs(vs.System.Environ())
	// if r.parent.Environment != nil {
	// 	r.parent.Environment.AddEnvs(vs.System.Environ())
	// }

	result := &api.Result{
		Value: b.String(),
	}
	args["result"] = result
	return result, nil
}

func (r *AgentScriptRunner) newExecHandler(vs *sh.VirtualSystem, envs api.ArgMap) sh.ExecHandler {
	return func(ctx context.Context, args []string) (bool, error) {
		if r.parent == nil {
			return true, fmt.Errorf("missing parent agent")
		}
		log.GetLogger(ctx).Debugf("parent: %s args: %+v\n", r.parent.Name, args)

		if conf.IsAction(strings.ToLower(args[0])) {
			log.GetLogger(ctx).Debugf("running ai agent/tool: %+v\n", args)

			// ignore result.
			// execv prints to stdout/stderr
			// argm, err := conf.Parse(args)
			// if err != nil {
			// 	return true, err
			// }
			// maps.Copy(envs, argm)
			// _, err = r.execv(ctx, vs, envs)
			// if err != nil {
			// 	return true, err
			// }

			at, err := conf.Parse(args)
			if err != nil {
				return true, err
			}
			var in = make(map[string]any)

			maps.Copy(in, r.parent.Arguments)
			//
			maps.Copy(in, r.sw.Vars.Global.GetAllEnvs())
			maps.Copy(in, envs)
			maps.Copy(in, at)

			_, err = r.execv(ctx, vs, in)
			if err != nil {
				return true, err
			}

			return true, nil
		}

		// internal list
		allowed := []string{"env", "printenv"}
		if slices.Contains(allowed, args[0]) {
			out, err := doBashCustom(vs, args)
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
		out, err := atm.ExecCommand(ctx, r.sw.OS, r.sw.Vars, args[0], args[1:])
		// out already has stdout/stder combined
		fmt.Fprintf(vs.IOE.Stdout, "%v", out)
		return true, err
	}
}

func (r *AgentScriptRunner) execv(ctx context.Context, vs *sh.VirtualSystem, args api.ArgMap) (*api.Result, error) {
	// for k, v := range r.parent.Environment.GetAllEnvs() {
	// 	vs.System.Setenv(k, v)
	// }

	result, err := r.sw.execm(ctx, r.parent, args)
	if result != nil {
		fmt.Fprintln(vs.IOE.Stdout, result.Value)
	}
	if err != nil {
		fmt.Fprintln(vs.IOE.Stderr, err.Error())
		return nil, err
	}
	return result, nil
}

func doBashCustom(vs *sh.VirtualSystem, args []string) (string, error) {
	switch args[0] {
	case "env", "printenv":
		var envs []string
		for k, v := range vs.System.Environ() {
			envs = append(envs, fmt.Sprintf("%s=%v", k, v))
		}
		return strings.Join(envs, "\n"), nil
	default:
	}
	return "", nil
}
