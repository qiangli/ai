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

// func (r *AgentScriptRunner) CreatorFrom(pack string, data []byte) (api.Creator, error) {
// 	// data, err := r.sw.Workspace.ReadFile(api.ToString(file), nil)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// extract pack name

// 	return r.sw.agentMaker.Creator(r.sw.agentMaker.Create, r.sw.User.Email, pack, data)
// }

// Run command or script. if script is empty, read command or script from args.
func (r *AgentScriptRunner) Run(ctx context.Context, script string, args map[string]any) (any, error) {
	if script == "" && args != nil {
		if c, ok := args["command"]; ok {
			script = api.ToString(c)
		} else {
			if v, err := r.sw.LoadScript(args); err == nil {
				script = v
			}
		}
	}

	if script == "" {
		return "", fmt.Errorf("missing bash command/script")
	}

	// // action
	// if name != "" {
	// 	if strings.Contains(name, ":") {
	// 		// run tool
	// 		kit, name := api.Kitname(name).Decode()
	// 		args["config"] = "data:" + script
	// 		args["kit"] = kit
	// 		args["name"] = name
	// 		return r.sw.Execm(ctx, args)
	// 	} else {
	// 		pack, _ := api.Packname(name).Decode()
	// 		creator, err := r.sw.agentMaker.Creator(r.sw.agentMaker.Create, r.sw.User.Email, pack, []byte(script))
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		return r.sw.runc(ctx, creator, r.parent, name, args)
	// 	}
	// 	// return nil, fmt.Errorf("invalid action: %s", name)
	// }

	// bash script
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
	return func(ctx context.Context, args []string) (bool, error) {
		if r.parent == nil {
			return true, fmt.Errorf("missing parent agent")
		}
		log.GetLogger(ctx).Debugf("parent: %s args: %+v\n", r.parent.Name, args)

		if conf.IsAction(strings.ToLower(args[0])) {
			log.GetLogger(ctx).Debugf("running ai agent/tool: %+v\n", args)

			_, err := r.execv(ctx, vs, args)
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

func (r *AgentScriptRunner) execv(ctx context.Context, vs *sh.VirtualSystem, args []string) (*api.Result, error) {
	for k, v := range r.parent.Environment.GetAllEnvs() {
		vs.System.Setenv(k, v)
	}

	result, err := r.sw.Execv(ctx, args)
	if err != nil {
		fmt.Fprintln(vs.IOE.Stderr, err.Error())
		return nil, err
	}
	fmt.Fprintln(vs.IOE.Stdout, result.Value)

	return result, nil
}

func (r *AgentScriptRunner) runc(ctx context.Context, creator api.Creator, vs *sh.VirtualSystem, name string, args map[string]any) (*api.Result, error) {
	for k, v := range r.parent.Environment.GetAllEnvs() {
		vs.System.Setenv(k, v)
	}

	result, err := r.sw.runc(ctx, creator, r.parent, name, args)

	if err != nil {
		fmt.Fprintln(vs.IOE.Stderr, err.Error())
		return nil, err
	}
	fmt.Fprintln(vs.IOE.Stdout, result.Value)

	return result, nil
}
