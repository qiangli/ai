package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	docli "github.com/qiangli/ai/swarm/faas"
)

type FaasKit struct {
}

func NewFaasKit() *FaasKit {
	return &FaasKit{}
}

func (r *FaasKit) Call(ctx context.Context, vars *api.Vars, token api.SecretToken, tf *api.ToolFunc, args map[string]any) (any, error) {
	tk, err := token()
	if err != nil {
		return nil, err
	}
	cli := docli.NewDoClient(tf.BaseUrl, tk)
	return cli.Call(ctx, vars, tf, args)
}
