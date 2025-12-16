package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
)

type SystemKit struct {
}

func NewSystemKit() *SystemKit {
	return &SystemKit{}
}

func (r *SystemKit) Call(ctx context.Context, vars *api.Vars, _ *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	v, err := CallKit(r, tf.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s:%s error: %w", tf.Kit, tf.Name, err)
	}
	return v, err
}
