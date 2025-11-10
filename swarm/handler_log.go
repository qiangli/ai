package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// MaxLogMiddleware returns a [api.Middleware] that logs the request and response
// for debugging. trim text to max length of n.
func MaxLogMiddlewareFunc(sw *Swarm) func(*api.Agent, int) api.Middleware {
	return func(agent *api.Agent, n int) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				logger := log.GetLogger(req.Context())
				// logger.Debugf("🟦 (log) request: %+v\n", req)
				logger.Debugf("🔗 (log): %s log_level: %v req: %+v\n", agent.Name, agent.LogLevel, req)

				err := next.Serve(req, resp)

				logger.Debugf("🔗 (log): %s resp: %+v\n", agent.Name, resp)
				if resp.Messages != nil {
					for _, m := range resp.Messages {
						logger.Debugf("%s %s\n", m.Role, clip(m.Content, n))
					}
				}
				return err
			})
		}
	}
}
