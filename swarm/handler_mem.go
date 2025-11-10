package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// MemoryMiddlewareFunc manages the loading/saving messages
func MemoryMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				ctx := req.Context()

				logger := log.GetLogger(ctx)

				logger.Debugf("🔗 (mem): %s\n", agent.Name)

				history, err := sw.History.Load(&api.MemOption{
					MaxHistory: agent.MaxHistory,
					MaxSpan:    agent.MaxSpan,
				})
				if err != nil {
					return err
				}

				initLen := len(history)
				sw.Vars.History = history

				logger.Debugf("init messages: %v\n", initLen)

				err = next.Serve(req, resp)

				nlen := (len(sw.Vars.History) - initLen)
				logger.Debugf("new messages: %v\n", nlen)

				if nlen > 0 {
					if err := sw.History.Save(sw.Vars.History[initLen:]); err != nil {
						logger.Errorf("error saving history: %v", err)
					}
					logger.Debugf("saved messages: %v\n", nlen)
				}

				return err
			})
		}
	}
}
