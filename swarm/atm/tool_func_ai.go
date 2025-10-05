package atm

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

type ListAgentsCacheKey struct {
	User string
}

var (
	listAgentsCache = expirable.NewLRU[ListAgentsCacheKey, string](10000, nil, time.Second*900)
)

func (r *FuncKit) ListAgents(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
	var user = r.user.Email
	// cached list
	key := ListAgentsCacheKey{
		User: user,
	}
	if v, ok := listAgentsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached agent list: %+v", key)
		return v, nil
	}

	list, count, err := conf.ListAgents(r.assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available agents: %v\n\n%s\n", count, list)
	listAgentsCache.Add(key, v)

	return v, nil
}

func (r *FuncKit) AgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	const tpl = `
Agent: %s
Display: %s
Description: %s
Instruction: %s
`

	agent, err := GetStrProp("agent", args)
	if err != nil {
		return "", err
	}
	ac, err := r.assets.FindAgent(r.user.Email, agent)
	if err != nil {
		return "", err
	}

	if ac != nil {
		for _, v := range ac.Agents {
			if v.Name == agent {
				return fmt.Sprintf(tpl, v.Name, v.Display, v.Description, clip(v.Instruction.Content, 100)), nil
			}
		}
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

func (r *FuncKit) AgentTransfer(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}
