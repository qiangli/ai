package swarm

import (
	"context"

	"github.com/qiangli/ai/internal/api"
)

func transferAgent(ctx context.Context, agent *Agent, name string, props map[string]any) (*Result, error) {
	transferTo, err := GetStrProp("agent", props)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		NextAgent: transferTo,
		State:     api.StateTransfer,
	}, nil
}
