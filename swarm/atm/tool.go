package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func NewToolCaller(auth *api.User, owner string, secrets api.SecretStore, tools api.ToolSystem) api.ToolCaller {

	toResult := func(v any) *api.Result {
		if r, ok := v.(*api.Result); ok {
			return r
		}
		if s, ok := v.(string); ok {
			return &api.Result{
				Value: s,
			}
		}
		return &api.Result{
			Value: fmt.Sprintf("%v", v),
		}
	}

	dispatch := func(ctx context.Context, vars *api.Vars, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
		kit, err := tools.GetKit(v.Type)
		if err != nil {
			return nil, err
		}
		token := func() (string, error) {
			return secrets.Get(owner, v.ApiKey)
		}
		out, err := kit.Call(ctx, vars, token, v, args)
		if err != nil {
			return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
		}
		return toResult(out), nil
	}

	return func(vars *api.Vars, agent *api.Agent) func(context.Context, string, map[string]any) (*api.Result, error) {
		toolMap := make(map[string]*api.ToolFunc)
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}

		return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
			v, ok := toolMap[tid]
			if !ok {
				return nil, fmt.Errorf("tool not found: %s", tid)
			}

			log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", v.Kit, v.Name, args)

			result, err := dispatch(ctx, vars, v, args)

			if err != nil {
				log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
			} else {
				log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
			}

			return result, err
		}
	}
}
