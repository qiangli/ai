package swarm

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

// hitorical context/memory
func ContextMiddleware(sw *Swarm) api.Middleware {
	mustResolveContext := func(parent *api.Agent, req *api.Request, s string) ([]*api.Message, error) {
		if !conf.IsAgentTool(s) {
			return nil, fmt.Errorf("invalid agent: %s", s)
		}
		out, err := sw.expandx(req.Context(), parent, s)
		if err != nil {
			return nil, err
		}
		if out == "" {
			return nil, nil
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

			var c = agent.Context
			if c != "" {
				if resolved, err := mustResolveContext(agent, req, c); err != nil {
					logger.Errorf("failed to resolve context %s: %v\n", c, err)
				} else {
					history = resolved
				}
			} else {
				opt := req.MemOption()
				if v, err := sw.History.Load(opt); err != nil {
					return err
				} else {
					history = v
				}
			}

			sw.Vars.SetHistory(history)

			err := next.Serve(req, resp)

			if v := sw.Vars.ListHistory(); len(v) > 0 {
				sw.History.Save(v)
			}

			return err
		})
	}
}
