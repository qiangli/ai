package swarm

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/shell/tool/sh/vfs"
)

// TODO rename and move to api?
type ToolFS struct {
	ws vfs.Workspace
}

func NewToolFS(ws vfs.Workspace) *ToolFS {
	return &ToolFS{
		ws: ws,
	}
}
func (r *ToolFS) Open(s string) (fs.File, error) {
	return r.ws.OpenFile(s, os.O_RDWR, 0o755)
}

type AgentToolRunner struct {
	sw      *Swarm
	agent   *api.Agent
	toolMap map[string]*api.ToolFunc
}

func NewAgentToolRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	toolMap := sw.buildAgentToolMap(agent)
	return &AgentToolRunner{
		sw:      sw,
		agent:   agent,
		toolMap: toolMap,
	}
}

func (r *AgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	v, ok := r.toolMap[tid]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", tid)
	}

	result, err := r.sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, r.sw.User), r.agent, v, args)
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

type AIAgentToolRunner struct {
	sw    *Swarm
	agent *api.Agent
}

func NewAIAgentToolRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	return &AIAgentToolRunner{
		sw:    sw,
		agent: agent,
	}
}

func (r *AIAgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	tools, err := conf.LoadToolFunc(r.agent.Owner, tid, r.sw.Secrets, r.sw.Assets)
	if err != nil {
		return nil, err
	}
	for _, v := range tools {
		id := v.ID()
		if id == tid {
			return r.sw.callTool(ctx, r.agent, v, args)
		}
	}
	return nil, fmt.Errorf("invalid tool: %s", tid)
}
