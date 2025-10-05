package atm

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sort"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

type FuncKit struct {
	user   *api.User
	assets api.AssetManager
}

func NewFuncKit(user *api.User, assets api.AssetManager) *FuncKit {
	return &FuncKit{
		user:   user,
		assets: assets,
	}
}

func (r *FuncKit) ListAgents(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
	var list []string

	dict, err := r.assets.ListAgent(r.user.Email)
	if err != nil {
		return "", err
	}

	for k, v := range dict {
		var desc []string
		for _, a := range v.Agents {
			desc = append(desc, a.Description)
		}
		list = append(list, fmt.Sprintf("%s: %s", k, strings.Join(desc, " ")))
	}
	sort.Strings(list)

	return fmt.Sprintf("Available agents:\n%s\n", strings.Join(list, "\n")), nil
}

func (r *FuncKit) AgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return "", err
	}
	ac, err := r.assets.FindAgent(r.user.Email, agent)
	if err != nil {
		return "", err
	}
	if ac != nil {
		var desc []string
		for _, a := range ac.Agents {
			desc = append(desc, a.Description)
		}
		return fmt.Sprintf("Agent: %s\nDescription: %s\n", ac.Name, strings.Join(desc, " ")), nil
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

func (r *FuncKit) Call(ctx context.Context, vars *api.Vars, token api.SecretToken, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return CallKit(r, tf.Config.Kit, tf.Name, callArgs...)
}
