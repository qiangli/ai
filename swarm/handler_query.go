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

			msg := agent.Message()
			if msg != "" {
				content, err := applyGlobal(agent.Template, "", msg, env)
				if err != nil {
					return err
				}

				// dynamic @agent
				content, err = resolve(agent, req, content)
				if err != nil {
					return err
				}

				req.SetQuery(content)
			}

			query := req.Query()
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
