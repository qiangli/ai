package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// User role query
func QueryMiddleware(sw *Swarm) api.Middleware {

	resolve := func(parent *api.Agent, req *api.Request, s string) (string, error) {
		at, found := parseAgentCommand(s)
		if !found {
			return s, nil
		}
		out, err := sw.callAgent(parent, req, at.Name, at.Message)
		if err != nil {
			return "", err
		}

		return out, nil
	}

	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())
			logger.Debugf("🔗 (query): %s\n", agent.Name)
			env := sw.globalEnv()

			if agent.Message != "" {
				content, err := applyGlobal(agent.Template, "", agent.Message, env)
				if err != nil {
					return err
				}

				// dynamic @agent
				content, err = resolve(agent, req, content)
				if err != nil {
					return err
				}

				req.Query = content
			}

			logger.Debugf("query: %s (%v)\n", abbreviate(req.Query, 64), len(req.Query))
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
