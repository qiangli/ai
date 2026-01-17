package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
)

type AgentToolRunner struct {
	user    string
	sw      *Swarm
	agent   *api.Agent
	toolMap map[string]*api.ToolFunc
}

func NewAgentToolRunner(sw *Swarm, user string, agent *api.Agent) api.ActionRunner {
	toolMap := sw.buildAgentToolMap(agent)
	return &AgentToolRunner{
		sw:      sw,
		user:    user,
		agent:   agent,
		toolMap: toolMap,
	}
}

func (r *AgentToolRunner) loadYaml(tid string, base string, script string) (*api.ToolFunc, error) {
	kit, name := api.Kitname(tid).Decode()
	// /agent:
	if kit == string(api.ToolTypeAgent) {
		ac, err := conf.LoadAgentsData([][]byte{[]byte(script)})
		if err != nil {
			return nil, err
		}
		pack, sub := api.Packname(name).Decode()
		if ac.Pack != pack {
			return nil, fmt.Errorf("Invalid pack name: %s. config: %s", pack, ac.Pack)
		}
		ac.Store = r.sw.Vars.RTE.Workspace
		ac.BaseDir = base
		for _, v := range ac.Agents {
			if (sub == "" && v.Name == pack) || sub == v.Name {
				v, err := conf.LoadAgentTool(ac, v.Name)
				if err == nil {
					v.Config = ac
					return v, nil
				}
			}
		}
	} else {
		// /kit:tool
		tc, err := conf.LoadToolData([][]byte{[]byte(script)})
		if err != nil {
			return nil, err
		}
		tc.Store = r.sw.Vars.RTE.Workspace
		tc.BaseDir = base
		tools, err := conf.LoadTools(tc, r.user, r.sw.Secrets)
		if err != nil {
			return nil, err
		}
		for _, v := range tools {
			if v.Kit == kit && v.Name == name {
				v.Config = tc
				return v, nil
			}
			// default
			if v.Kit == kit && name == "" && v.Name == kit {
				v.Config = tc
				return v, nil
			}
		}
	}
	return nil, nil
}

func (r *AgentToolRunner) loadScript(uri string) (string, error) {
	if strings.HasPrefix(uri, "data:") {
		return api.DecodeDataURL(uri)
	} else {
		var f = uri
		if strings.HasPrefix(f, "file:") {
			v, err := url.Parse(f)
			if err != nil {
				return "", err
			}
			f = v.Path
		}
		data, err := r.sw.Workspace.ReadFile(f, nil)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

// try load from provided script file first,
// if not found, continue to load from other sources
func (r *AgentToolRunner) loadTool(tid string, args map[string]any) (*api.ToolFunc, error) {
	// load tool from content
	if s, ok := args["script"]; ok {
		s := api.ToString(s)
		ext := path.Ext(s)
		switch ext {
		case ".sh", ".bash":
			// continue
		case ".yaml", ".yml":
			cfg, err := r.loadScript(s)
			if err != nil {
				return nil, err
			}
			base := filepath.Dir(s)
			tf, err := r.loadYaml(tid, base, cfg)
			if err != nil {
				return nil, err
			}
			if tf != nil {
				return tf, nil
			}
			// return nil, fmt.Errorf("%q not found in script: %s", tid, s)
			// continue to load tid
		case ".txt", ".md", ".markdown":
			// feed text file as query content
			cfg, err := r.loadScript(s)
			if err != nil {
				return nil, err
			}
			old := args["content"]
			args["content"] = api.Cat(api.ToString(old), tail(cfg, 1), "\n###\n")
			// continue to load tid
		case ".json", ".jsonc":
			// decode json as additional arguments
			cfg, err := r.loadScript(s)
			if err != nil {
				return nil, err
			}
			obj := tail(cfg, 1)
			obj = strings.TrimSpace(obj)
			if err := json.Unmarshal([]byte(obj), &args); err != nil {
				return nil, err
			}
			// continue to load tid
		default:
			// ignore script property
		}
	}

	// inline
	v, ok := r.toolMap[tid]
	if ok {
		return v, nil
	}

	// load external
	tools, err := conf.LoadToolFunc(r.user, tid, r.sw.Secrets, r.sw.Assets)
	if err != nil {
		return nil, err
	}
	for _, v := range tools {
		id := v.ID()
		if id == tid {
			return v, nil
		}
	}
	return nil, fmt.Errorf("invalid tool: %s", tid)
}

// Lookup aciton for tid or kit:name in the args if tid is not provided
func (r *AgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	var kit, name string
	if tid == "" {
		kit = api.ToString(args["kit"])
		name = api.ToString(args["name"])
		if kit == "agent" {
			pack := api.ToString(args["pack"])
			if pack == "" {
				return nil, fmt.Errorf("missing pack for agent tool: %s", name)
			}
			name = pack + "/" + name
		}
		tid = api.NewKitname(kit, name).ID()
	} else {
		kit, name = api.Kitname(tid).Decode()
	}

	// /bin/command (local system)
	// sh:*
	// agent:*
	// kit:*

	// default tool
	// if name == "" {
	// 	name = kit
	// }

	// /alias:name --option name="command line"
	// name to lookup the command in args
	// TODO alias referencing alias
	// alias use the same args in the same scope
	if kit == string(api.ToolTypeAlias) {
		if name == "" {
			return nil, fmt.Errorf("Invalid alias action. missing name. /alias:NAME --option NAME='action to run'")
		}

		// TODO expand to general action not just for alias
		// execute if action runner is provided
		if v, found := args[name]; found {
			if runner, ok := v.(api.ActionRunner); ok {
				return runner.Run(ctx, tid, args)
			}
		}

		//
		alias, err := api.GetStrProp(name, args)
		if err != nil {
			return nil, fmt.Errorf("Failed to resolve alias: %s, error: %v", name, err)
		}
		if alias == "" {
			return nil, fmt.Errorf("Alias not found: %s", name)
		}

		argv := conf.Argv(alias)
		if len(argv) == 0 {
			return nil, fmt.Errorf("Invalid alias %q: %s", name, alias)
		}
		//
		if conf.IsAction(argv[0]) {
			nargs, err := conf.Parse(argv)
			if err != nil {
				return nil, err
			}
			maps.Copy(args, nargs)
			kit, name = api.Kitname(argv[0]).Decode()
		} else {
			// exec alias as system command line
			kit = string(api.ToolTypeBin)
			args["command"] = alias
		}
	}

	// this ensures kit:name is in internal kit__name format
	tid = api.NewKitname(kit, name).ID()

	var tf *api.ToolFunc

	// TODO this shortcut save time for a few function calls.
	// reinstate if performance is hit
	// if kit == string(api.ToolTypeAgent) {
	// 	return r.sw.runm(ctx, r.agent, name, args)
	// }

	// system command
	// /bin/*
	// /bin: pseudo kit name "bin"
	// support direct execution of system command using the standad syntax
	// e.g. /bin/ls without the trigger word "ai"
	// and slash command toolkit "/kit:name" syntax
	// i.e. equivalent to: /sh:exec --arg command="ls -al /tmp"
	// TODO kit required?
	if kit == "" || kit == string(api.ToolTypeBin) {
		cmd, _ := api.GetStrProp("command", args)
		if cmd == "" {
			return nil, fmt.Errorf("missing system command")
		}
		// return atm.ExecCommand(ctx, r.sw.OS, r.sw.Vars, cmd, nil)
		tf = &api.ToolFunc{
			Type: api.ToolTypeBin,
			Kit:  string(api.ToolTypeBin),
			Name: cmd,
		}
	} else {
		// agent/tool action
		v, err := r.loadTool(tid, args)
		if err != nil {
			return nil, err
		}
		tf = v
	}

	// // NOTE: should this be done?
	// // set if only not existing
	// envs := r.sw.Vars.Global.GetAllEnvs()
	// for k, v := range envs {
	// 	if _, ok := args[k]; !ok {
	// 		args[k] = v
	// 	}
	// }

	// run the action
	// and make the error/result available in args
	uctx := context.WithValue(ctx, api.SwarmUserContextKey, r.user)
	result, err := r.sw.callTool(uctx, r.agent, tf, args)

	if err != nil {
		// args["error"] = err.Error()
		return "", err
	}
	if result == nil {
		result = &api.Result{}
	}
	// args["result"] = result
	return result, err
}
