package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// ModelMiddlewareFunc loads the dynamical modle
func ModelMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				ctx := req.Context()
				log.GetLogger(ctx).Debugf("🔗 (model): %s model: %s\n", agent.Name, agent.Model)

				var model *api.Model = agent.Model

				// resolve if model is @agent
				if v, err := sw.resolveModel(agent, ctx, req, agent.Model); err != nil {
					return err
				} else {
					model = v
				}

				//
				ak, err := sw.Secrets.Get(agent.Owner, model.ApiKey)
				if err != nil {
					return err
				}
				token := func() string {
					return ak
				}

				nreq := req.Clone()
				nreq.Model = model
				nreq.Token = token

				err = next.Serve(nreq, resp)

				return err
			})
		}
	}
}
