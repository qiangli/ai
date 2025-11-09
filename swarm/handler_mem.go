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

				history, err := sw.History.Load(&api.MemOption{
					MaxHistory: agent.MaxHistory,
					MaxSpan:    agent.MaxSpan,
				})
				if err != nil {
					return err
				}

				initLen := len(history)
				sw.Vars.History = history

				log.GetLogger(ctx).Debugf("Loading conversation messages: %v\n", initLen)

				err = next.Serve(req, resp)

				if len(sw.Vars.History) > initLen {
					log.GetLogger(ctx).Debugf("Saving conversation messages: %v\n", (len(sw.Vars.History) - initLen))
					if err := sw.History.Save(sw.Vars.History[initLen:]); err != nil {
						log.GetLogger(ctx).Errorf("error saving conversation history: %v", err)
					}
				}

				return err
			})
		}
	}
}
