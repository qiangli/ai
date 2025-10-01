package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
	docli "github.com/qiangli/ai/swarm/faas"
)

type FaasKit struct {
	BaseUrl string
	Token   string
}

func (r *FaasKit) callTool(ctx context.Context, vars *api.Vars, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	cli := docli.NewDoClient(r.BaseUrl, r.Token)
	return cli.Call(ctx, vars, tf, args)
}
