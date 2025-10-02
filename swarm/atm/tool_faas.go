package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	docli "github.com/qiangli/ai/swarm/faas"
)

type FaasKit struct {
	BaseUrl string
}

func (r *FaasKit) Call(ctx context.Context, vars *api.Vars, token api.SecretToken, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	tk, err := token()
	if err != nil {
		return nil, err
	}
	cli := docli.NewDoClient(r.BaseUrl, tk)
	return cli.Call(ctx, vars, tf, args)
}
