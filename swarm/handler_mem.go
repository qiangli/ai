package swarm

import (
	"encoding/json"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func MemoryMiddleware(sw *Swarm) api.Middleware {

	mustResolveContext := func(parent *api.Agent, req *api.Request, s string) ([]*api.Message, error) {
		out, err := sw.resolveCommand(parent, req, s)
		if err != nil {
			return nil, err
		}
		var list []*api.Message
		if err := json.Unmarshal([]byte(out), &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())

			logger.Debugf("🔗 (mem): %s\n", agent.Name)
			var history []*api.Message
			// var emoji = "•"
			// override if context agent is specified
			c := agent.Arguments.GetString("context")
			if c != "" {
				if resolved, err := mustResolveContext(agent, req, c); err != nil {
					logger.Errorf("failed to resolve context %s: %v\n", c, err)
				} else {
					history = resolved
					// emoji = "🤖"
				}
			}

			// history, err := sw.History.Load(&api.MemOption{
			// 	MaxHistory: agent.MaxHistory,
			// 	MaxSpan:    agent.MaxSpan,
			// })
			// if err != nil {
			// 	return err
			// }

			sw.Vars.SetHistory(history)

			// logger.Debugf("init messages: %v\n", len(history))

			err := next.Serve(req, resp)

			return err
		})
	}
}
