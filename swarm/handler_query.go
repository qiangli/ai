package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// User role query
func QueryMiddleware(sw *Swarm) api.Middleware {

	resolve := func(parent *api.Agent, req *api.Request, s string) (string, error) {
		out, err := sw.expand(req.Context(), parent, s)
		if err != nil {
			return "", err
		}
		return out, nil
	}

	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())
			logger.Debugf("🔗 (query): %s\n", agent.Name)

			var env = req.Arguments.GetAllArgs()

			// convert user message into query if not set
			query := req.Query()

			if query == "" {
				msg := agent.Message()
				if msg != "" {
					content, err := applyGlobal(agent.Template, msg, env)
					if err != nil {
						return err
					}

					// dynamic @agent
					content, err = resolve(agent, req, content)
					if err != nil {
						return err
					}

					req.SetQuery(content)
				} else {
					req.SetQuery(env[globalQuery])
				}
			}

			logger.Debugf("query: %s (%v)\n", abbreviate(query, 64), len(query))
			if logger.IsTrace() {
				logger.Debugf("query: %s\n", req.Query)
			}

			if err := next.Serve(req, resp); err != nil {
				return err
			}
			return nil
		})
	}
}
