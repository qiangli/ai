package atm

import (
	"context"

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

func (r *FuncKit) Call(ctx context.Context, vars *api.Vars, _ api.SecretToken, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return CallKit(r, tf.Config.Kit, tf.Name, callArgs...)
}
