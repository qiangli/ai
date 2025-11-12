package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// MaxLogMiddleware returns a [api.Middleware] that logs the request and response
// for debugging. trim text to max length of n.
func MaxLogMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				logger := log.GetLogger(req.Context())
				logger.Debugf("🔗 (log): %s log_level: %v req: %+v\n", agent.Name, agent.LogLevel, req)

				err := next.Serve(req, resp)

				logger.Debugf("agent: %s resp: (%v)\n", agent.Name, len(resp.Messages))
				if resp.Messages != nil {
					for _, m := range resp.Messages {
						logger.Debugf("%s %s (%v)\n", m.Role, clip(m.Content, 64), len(m.Content))
						if logger.IsTrace() {
							logger.Debugf("%s\n", m.Content)
						}
					}
				}
				return err
			})
		}
	}
}
