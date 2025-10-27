package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/tool/memory"
)

type FuncKit struct {
	user   *api.User
	assets api.AssetManager
	kgm    *memory.KGManager
}

func NewFuncKit(user *api.User, assets api.AssetManager) *FuncKit {
	return &FuncKit{
		user:   user,
		assets: assets,
		kgm:    memory.NewKGManager(),
	}
}

func (r *FuncKit) Call(ctx context.Context, vars *api.Vars, _ *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return CallKit(r, tf.Kit, tf.Name, callArgs...)
}
