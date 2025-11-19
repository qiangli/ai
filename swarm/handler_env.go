package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// EnvMiddleware process global environment variables
func EnvMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			env := sw.globalEnv()
			var args map[string]any
			if req.Arguments != nil {
				args = make(map[string]any)
				req.Arguments.Copy(args)
			}
			sw.mapAssign(agent, req, env, args, false)

			log.GetLogger(req.Context()).Debugf("🔗 (env): %s env: %+v\n", agent.Name, env)

			err := next.Serve(req, resp)

			return err
		})
	}
}
