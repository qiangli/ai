package swarm

import (
	"fmt"
	"maps"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// Init and start the chain
func InitEnvMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())

			// update envs and args
			envs := sw.globalEnv()

			// environment
			// var envs = make(map[string]any)
			// query/result/error
			// maps.Copy(envs, global)
			if agent.Environment != nil {
				agent.Environment.Copy(envs)
			}

			// args
			var args = make(map[string]any)
			if agent.Arguments != nil {
				agent.Arguments.Copy(args)
			}
			if req.Arguments != nil {
				req.Arguments.Copy(args)
			}
			// TODO better way for the cyclical ref for applying tempate
			// chick and egg. env reference to arg and arg to env
			//
			maps.Copy(envs, args)
			sw.mapAssign(agent, req, envs, envs, true)
			//
			sw.mapAssign(agent, req, args, envs, false)

			nreq := req.Clone()
			nreq.Arguments.SetArgs(args)

			ll := nreq.Arguments.GetString("log_level")
			logger.SetLogLevel(api.ToLogLevel(ll))

			var parent string
			if agent.Parent != nil {
				parent = fmt.Sprintf("%s → ", agent.Parent.Name)
			}
			logger.Infof("🚀 %s%s\n", parent, agent.Name)

			return next.Serve(nreq, resp)
		})
	}
}
