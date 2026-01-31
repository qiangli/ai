package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	docli "github.com/qiangli/ai/swarm/faas"
)

func (r *FuncKit) DO(ctx context.Context, vars *api.Vars, _ *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// python with digital ocean
	apiKey, _ := api.GetStrProp("api_key", args)
	baseUrl, _ := api.GetStrProp("base_url", args)

	tk, err := vars.Token(apiKey)
	if err != nil {
		return nil, err
	}
	cli := docli.NewDoClient(baseUrl, tk)
	return cli.Call(ctx, vars, tf, args)
}
