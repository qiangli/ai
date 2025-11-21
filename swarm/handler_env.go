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
			// envs
			var envs = make(map[string]any)
			maps.Copy(envs, sw.globalEnv())
			if agent.Environment != nil {
				sw.mapAssign(ctx, agent, envs, agent.Environment.GetAllEnvs(), true)
			}
			agent.Environment.AddEnvs(envs)

			// args
			// global envs
			// agent envs
			// agent args
			// req args
			var args = make(map[string]any)
			maps.Copy(args, envs)
			if agent.Arguments != nil {
				sw.mapAssign(ctx, agent, args, agent.Arguments.GetAllArgs(), true)
			}
			if req.Arguments != nil {
				sw.mapAssign(ctx, agent, args, req.Arguments.GetAllArgs(), true)
			}

			nreq := req.Clone()
			nreq.Arguments.SetArgs(args)

			ll := nreq.Arguments.GetString("log_level")
			logger.SetLogLevel(api.ToLogLevel(ll))

			var parent string
			if agent.Parent != nil {
				parent = fmt.Sprintf("%s → ", agent.Parent.Name)
			}
			logger.Infof("🚀 %s%s\n", parent, agent.Name)
			query := nreq.Arguments.Query()
			if query != "" {
				sw.Vars.Global.Set(globalQuery, query)
				logger.Infof("query: %v\n", parent, query)
			}

			return next.Serve(nreq, resp)
		})
	}
}
