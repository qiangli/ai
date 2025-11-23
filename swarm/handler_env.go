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

			var ctx = req.Context()

			// update envs and args
			//
			// envs
			var envs = make(map[string]any)
			maps.Copy(envs, sw.globalEnv())
			if agent.Environment != nil {
				aenvs := agent.Environment.GetAllEnvs()
				sw.mapAssign(ctx, agent, envs, aenvs, true)
			}
			agent.Environment.AddEnvs(envs)

			// args
			//
			// global/agent envs
			// agent args
			// req args
			var args = make(map[string]any)
			maps.Copy(args, envs)
			if agent.Arguments != nil {
				aargs := agent.Arguments.GetAllArgs()
				sw.mapAssign(ctx, agent, args, aargs, true)
			}
			if req.Arguments != nil {
				rargs := req.Arguments.GetAllArgs()
				sw.mapAssign(ctx, agent, args, rargs, true)
			}
			req.Arguments.SetArgs(args)

			ll := req.Arguments.GetString("log_level")
			logger.SetLogLevel(api.ToLogLevel(ll))

			var parent string
			if agent.Parent != nil {
				parent = fmt.Sprintf("%s → ", agent.Parent.Name)
			}
			logger.Infof("🚀 %s%s\n", parent, agent.Name)

			return next.Serve(req, resp)
		})
	}
}
