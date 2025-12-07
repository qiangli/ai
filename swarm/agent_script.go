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
	sw     *Swarm
	parent *api.Agent
}

func NewAgentScriptRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	return &AgentScriptRunner{
		sw:     sw,
		parent: agent,
	}
}

// Run command or script. if script is empty, read command or script from args.
func (r *AgentScriptRunner) Run(ctx context.Context, script string, args map[string]any) (any, error) {
	if script == "" && args != nil {
		if c, ok := args["command"]; ok {
			script = api.ToString(c)
		} else if file, ok := args["script"]; ok {
			data, err := r.sw.Workspace.ReadFile(api.ToString(file), nil)
			if err != nil {
				return "", err
			}
			script = string(data)
		}
	}
	if script == "" {
		return "", fmt.Errorf("missing bash command/script")
	}

	var b bytes.Buffer
	ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
	vs := sh.NewVirtualSystem(r.sw.Root, r.sw.OS, r.sw.Workspace, ioe)

	// set global env for bash script
	env := r.sw.globalEnv()

	// TODO batch set
	for k, v := range env {
		vs.System.Setenv(k, v)
	}

	vs.ExecHandler = r.newExecHandler(vs)

	if err := vs.RunScript(ctx, script); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (r *AgentScriptRunner) newExecHandler(vs *sh.VirtualSystem) sh.ExecHandler {
	var runner = r.runner(vs, r.parent)

	return func(ctx context.Context, args []string) (bool, error) {
		if r.parent == nil {
			return true, fmt.Errorf("missing parent agent")
		}
		log.GetLogger(ctx).Debugf("parent: %s args: %+v\n", r.parent.Name, args)
		// isAI := conf.IsAgentTool
		// isAI := func(s string) bool {
		// 	return s == "ai" || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "/")
		// }
		if conf.IsAction(strings.ToLower(args[0])) {
			log.GetLogger(ctx).Debugf("running ai agent/tool: %+v\n", args)

			_, err := runner(ctx, args)
			if err != nil {
				return true, err
			}
			return true, nil
		}

		// internal list
		allowed := []string{"env", "printenv"}
		if slices.Contains(allowed, args[0]) {
			doBashCustom(vs, args)
			return true, nil
		}

		// bash core utils
		if did, err := sh.RunCoreUtils(ctx, vs, args); did {
			return did, err
		}

		// TODO restricted
		// block other commands
		// fmt.Fprintf(vs.IOE.Stderr, "command not supported: %s %+v\n", args[0], args[1:])
		atm.ExecCommand(ctx, r.sw.OS, r.sw.Vars, args[0], args[1:])
		return true, nil
	}
}

func (r *AgentScriptRunner) runner(vs *sh.VirtualSystem, agent *api.Agent) func(context.Context, []string) (*api.Result, error) {
	return func(ctx context.Context, args []string) (*api.Result, error) {

		for k, v := range agent.Environment.GetAllEnvs() {
			vs.System.Setenv(k, v)
		}

		// at, err := conf.ParseActionArgs(args)
		// if err != nil {
		// 	return nil, err
		// }
		// id := at.Kitname().ID()
		// // TODO batch set
		// data, err := agent.Runner.Run(ctx, id, at)

		result, err := r.sw.Execv(ctx, args)

		if err != nil {
			// vs.System.Setenv(globalError, err.Error())
			fmt.Fprintln(vs.IOE.Stderr, err.Error())
			return nil, err
		}
		// result := api.ToResult(data)
		// if result == nil {
		// 	result = &api.Result{}
		// }
		fmt.Fprintln(vs.IOE.Stdout, result.Value)
		// vs.System.Setenv(globalResult, result.Value)

		return result, nil
	}
}
