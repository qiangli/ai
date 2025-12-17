package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
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

func (r *AgentToolRunner) loadYaml(tid string, script string) (*api.ToolFunc, error) {
	tc, err := conf.LoadToolData([][]byte{[]byte(script)})
	if err != nil {
		return nil, err
	}
	tools, err := conf.LoadTools(tc, r.user, r.sw.Secrets)
	if err == nil {
		return nil, err
	}
	kit, name := api.Kitname(tid).Decode()
	// /agent:
	if kit == string(api.ToolTypeAgent) {
		//
		// TODO load app config including both agents/tools
		// to replace this hack
		_, sub := api.Packname(name).Decode()
		for _, v := range tc.Agents {
			if sub == v.Name {
				v, err := conf.LoadAgentTool(tc, sub)
				if err == nil {
					v.Name = name
					return v, nil
				}
			}
		}
	} else {
		// /kit:tool
		for _, v := range tools {
			if v.Kit == kit && v.Name == name {
				return v, nil
			}
			// default
			if v.Kit == kit && name == "" && v.Name == kit {
				return v, nil
			}
		}
	}
	return nil, nil
}

func (r *AgentToolRunner) loadTool(tid string, args map[string]any) (*api.ToolFunc, error) {
	// load tool from content
	if s, ok := args["script"]; ok {
		s := api.ToString(s)
		ext := path.Ext(s)
		switch ext {
		case ".sh", ".bash":
			// continue
		case ".yaml", ".yml":
			cfg, err := r.sw.LoadScript(s)
			if err != nil {
				return nil, err
			}
			tf, _ := r.loadYaml(tid, cfg)
			if tf != nil {
				return tf, nil
			}
		case ".txt", ".md", ".markdown":
			// feed text file as query content
			cfg, err := r.sw.LoadScript(s)
			if err != nil {
				return nil, err
			}
			old := args["content"]
			args["content"] = api.Cat(api.ToString(old), tail(cfg, 1), "\n###\n")
			// continue to load tid
		case ".json", ".jsonc":
			// decode json into additional arguments
			cfg, err := r.sw.LoadScript(s)
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

func (r *AgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	var kit, name string
	if tid == "" {
		kit = api.ToString(args["kit"])
		name = api.ToString(args["name"])
		tid = api.Kitname(kit + ":" + name).ID()
	} else {
		kit, name = api.Kitname(tid).Decode()
	}

	// /bin/command (local system)
	// sh:*
	// agent:*
	// kit:*

	// default tool
	if name == "" {
		name = kit
	}
	// this ensures kit:name is in internal kit__name format
	tid = api.Kitname(kit + ":" + name).ID()

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
	if kit == "" || kit == "bin" {
		cmd, _ := api.GetStrProp("command", args)
		return atm.ExecCommand(ctx, r.sw.OS, r.sw.Vars, cmd, nil)
	}

	// agent/tool action
	v, err := r.loadTool(tid, args)
	if err != nil {
		return nil, err
	}

	// clear
	delete(args, "result")
	delete(args, "error")

	result, err := r.sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, r.user), r.agent, v, args)

	if err != nil {
		args["error"] = err.Error()
	}
	if result != nil {
		args["result"] = result.Value
	}
	// // log calls
	// r.sw.Vars.AddToolCall(&api.ToolCallEntry{
	// 	ID:        tid,
	// 	Kit:       v.Kit,
	// 	Name:      v.Name,
	// 	Arguments: v.Arguments,
	// 	Result:    result,
	// 	Error:     err,
	// 	Timestamp: time.Now(),
	// })
	return result, err
}
