package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// Init and start the chain
func InitEnvMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())

			var args = make(map[string]any)
			if agent.Arguments != nil {
				agent.Arguments.Copy(args)
			}
			if req.Arguments != nil {
				req.Arguments.Copy(args)
			}

			env := sw.globalEnv()
			sw.mapAssign(agent, req, env, args, false)

			nreq := req.Clone()
			nreq.Arguments.SetArgs(args)

			ll := nreq.Arguments.GetString("log_level")
			logger.SetLogLevel(api.ToLogLevel(ll))

			logger.Debugf("🔗 (init): %s\n", agent.Name)

			logger.Infof("🚀 %s ← %s\n", agent.Name, NilSafe(agent.Parent).Name)

			return next.Serve(nreq, resp)
		})
	}
}
