package swarm

import (
	"context"
	"fmt"
	// "io/fs"
	// "os"
	"time"

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

func (r *AgentToolRunner) loadTool(tid string, args map[string]any) (*api.ToolFunc, error) {
	// inline
	v, ok := r.toolMap[tid]
	if ok {
		return v, nil
	}

	// load from content
	cfg, err := r.sw.LoadActionConfig(args)
	if err == nil {
		tc, err := conf.LoadToolData([][]byte{[]byte(cfg)})
		if err == nil {
			tools, err := conf.LoadTools(tc, r.user, r.sw.Secrets)
			if err == nil {
				for _, v := range tools {
					if v.ID() == tid {
						return v, nil
					}
				}
			}
		}
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
	kit, name := api.Kitname(tid).Decode()
	// local system command
	// sh:*

	// default tool
	if name == "" {
		name = kit
	}
	if kit == "" || kit == "sh" {
		// system command
		if kit == "" {
			cmd, _ := api.GetStrProp("command", args)
			argv, _ := api.GetArrayProp("arguments", args)
			return atm.ExecCommand(ctx, r.sw.OS, r.sw.Vars, cmd, argv)
		}
		// shell
		return r.agent.Shell.Run(ctx, "", args)
	}

	// agent/tool action
	v, err := r.loadTool(tid, args)
	if err != nil {
		return nil, err
	}

	result, err := r.sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, r.user), r.agent, v, args)

	// log calls
	r.sw.Vars.AddToolCall(&api.ToolCallEntry{
		ID:        tid,
		Kit:       v.Kit,
		Name:      v.Name,
		Arguments: v.Arguments,
		Result:    result,
		Error:     err,
		Timestamp: time.Now(),
	})
	return result, err
}
