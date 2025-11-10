package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// EnvMiddlewareFunc process global environment variables
func EnvMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				nreq := req.Clone()
				ctx := nreq.Context()
				env := globalEnv(sw)

				log.GetLogger(ctx).Debugf("🟦 (env): %s %+v\n", agent.Name, env)

				mapAssign(sw, agent, req, env, req.Arguments, false)

				err := next.Serve(nreq, resp)

				return err
			})
		}
	}
}
