package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	docli "github.com/qiangli/ai/swarm/faas"
)

type FaasKit struct {
	secrets api.SecretStore
}

func NewFaasKit(secrets api.SecretStore) *FaasKit {
	return &FaasKit{
		secrets: secrets,
	}
}

func (r *FaasKit) Call(ctx context.Context, vars *api.Vars, env *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	tk, err := r.secrets.Get(env.Owner, tf.ApiKey)
	if err != nil {
		return nil, err
	}
	cli := docli.NewDoClient(tf.BaseUrl, tk)
	return cli.Call(ctx, vars, tf, args)
}
