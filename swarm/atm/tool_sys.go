package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
)

type SystemKit struct {
	git *GitKit
}

func NewSystemKit() *SystemKit {
	return &SystemKit{
		git: &GitKit{},
	}
}

func (r *SystemKit) Call(ctx context.Context, vars *api.Vars, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// dispatch git:*
	if tf.Kit == "git" {
		return r.git.Call(ctx, vars, agent, tf, args)
	}

	// TODO refactor
	callArgs := []any{ctx, vars, tf.Name, args}
	v, err := CallKit(r, tf.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s:%s error: %w", tf.Kit, tf.Name, err)
	}
	return v, err
}
