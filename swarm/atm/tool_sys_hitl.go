package atm

import (
	"context"

	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/swarm/api"
)

func (r *SystemKit) Confirm(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	prompt, err := api.GetStrProp("prompt", args)
	if err != nil {
		return "", err
	}
	return bubble.Confirm(prompt)
}
