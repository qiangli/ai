package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func ToolMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				logger := log.GetLogger(req.Context())

				logger.Debugf("🔗 (tool): %s\n", agent.Name)

				var tools = make(map[string]*api.ToolFunc)

				// inherit tools of embeeded agents
				if agent.Embed != nil {
					var addAll func(*api.Agent) error
					addAll = func(a *api.Agent) error {
						if a.Embed != nil {
							for _, v := range a.Embed {
								return addAll(v)
							}
						}
						for _, v := range a.Tools {
							tools[v.ID()] = v
						}
						return nil
					}

					addAll(agent)

					var list []*api.ToolFunc
					for _, v := range tools {
						list = append(list, v)
					}
					agent.Tools = list
				}

				return next.Serve(req, resp)
			})
		}
	}
}
