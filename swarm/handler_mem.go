package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// MemoryMiddleware manages the loading/saving messages
func MemoryMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())

			logger.Debugf("🔗 (mem): %s\n", agent.Name)

			history, err := sw.History.Load(&api.MemOption{
				MaxHistory: agent.MaxHistory,
				MaxSpan:    agent.MaxSpan,
			})
			if err != nil {
				return err
			}

			// initLen := len(history)
			sw.Vars.InitHistory(history)

			logger.Debugf("init messages: %v\n", len(history))

			err = next.Serve(req, resp)

			nhist := sw.Vars.GetNewHistory()
			nlen := len(nhist)
			// nlen := (len(sw.Vars.History) - initLen)
			logger.Debugf("new messages: %v\n", nlen)

			if nlen > 0 {
				if err := sw.History.Save(nhist); err != nil {
					logger.Errorf("error saving history: %v", err)
				}
				logger.Debugf("saved messages: %v\n", nlen)
			}

			return err
		})
	}
}
