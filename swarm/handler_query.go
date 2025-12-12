package swarm

import (
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

// User role query
func QueryMiddleware(sw *Swarm) api.Middleware {

	resolve := func(parent *api.Agent, req *api.Request, s string) (string, error) {
		if !conf.IsAction(s) {
			return s, nil
		}
		out, err := sw.expandx(req.Context(), parent, s)
		if err != nil {
			return "", err
		}
		return out, nil
	}

	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())
			logger.Debugf("🔗 (query): %s\n", agent.Name)

			var allArgs = req.Arguments.GetAllArgs()

			// convert user message into query if not set
			// var query = agent.Query()
			var query = req.Query
			if query == "" {
				msg := req.Message()
				if msg != "" {
					content, err := atm.CheckApplyTemplate(agent.Template, msg, allArgs)
					if err != nil {
						return err
					}

					// dynamic @agent
					content, err = resolve(agent, req, content)
					if err != nil {
						return err
					}

					query = content
				} else {
					return fmt.Errorf("no input message")
				}
			}

			//
			if agent.Message != "" {
				query = agent.Message + "\n" + query
			}
			// agent.SetQuery(query)
			req.Query = query

			logger.Debugf("query: %s (%v)\n", abbreviate(query, 64), len(query))
			if logger.IsTrace() {
				logger.Debugf("query: %s\n", query)
			}

			if err := next.Serve(req, resp); err != nil {
				return err
			}
			return nil
		})
	}
}
