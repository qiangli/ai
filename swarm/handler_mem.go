package swarm

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func MemoryMiddleware(sw *Swarm) api.Middleware {

	mustResolveContext := func(parent *api.Agent, req *api.Request, s string) ([]*api.Message, error) {
		at, found := parseAgentCommand(s)
		if !found {
			return nil, fmt.Errorf("invalid context: %s", s)
		}
		nreq := req.Clone()
		if len(at.Arguments) > 0 {
			if nreq.Arguments == nil {
				at.Arguments = make(map[string]any)
			}
			maps.Copy(nreq.Arguments, at.Arguments)
		}
		out, err := sw.callAgent(parent, nreq, at.Name, at.Message)
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
			if agent.Context != "" {
				if resolved, err := mustResolveContext(agent, req, agent.Context); err != nil {
					logger.Errorf("failed to resolve context %s: %v\n", agent.Context, err)
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
