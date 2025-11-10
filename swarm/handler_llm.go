package swarm

import (
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

// InferenceMiddlewareFunc loads the dynamical modle
func InferenceMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				ctx := req.Context()
				log.GetLogger(ctx).Debugf("🔗 (llm): %s adapter: %s\n", agent.Name, agent.Adapter)

				var adapter llm.LLMAdapter = adapter.Chat
				if agent.Adapter != "" {
					if v, err := sw.Adapters.Get(agent.Adapter); err == nil {
						adapter = v
					} else {
						return err
					}
				}

				// LLM adapter
				// TODO model <-> adapter
				result, err := adapter(ctx, req)
				if err != nil {
					return err
				}

				if result.Result == nil {
					return fmt.Errorf("Empty response")
				}
				resp.Result = result.Result

				return next.Serve(req, resp)
			})
		}
	}
}
