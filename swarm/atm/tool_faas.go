package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	docli "github.com/qiangli/ai/swarm/faas"
)

func (r *FuncKit) DO(ctx context.Context, vars *api.Vars, env *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	// python with digital ocean
	tk, err := vars.Token(tf.ApiKey)
	if err != nil {
		return nil, err
	}
	cli := docli.NewDoClient(tf.BaseUrl, tk)
	return cli.Call(ctx, vars, tf, args)
}
