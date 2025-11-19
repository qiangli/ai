package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

// InferenceMiddleware loads the dynamical modle
func InferenceMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			ctx := req.Context()
			logger := log.GetLogger(ctx)
			logger.Debugf("🔗 (llm): %s adapter: %s\n", agent.Name, agent.Adapter)

			var adapter llm.LLMAdapter = &adapter.ChatAdapter{}
			if agent.Adapter != "" {
				if v, err := sw.Adapters.Get(agent.Adapter); err == nil {
					adapter = v
				} else {
					return err
				}
			}

			// LLM adapter
			// TODO model <-> adapter
			result, err := adapter.Call(ctx, req)
			if err != nil {
				return err
			}

			if result.Result == nil {
				result.Result = &api.Result{
					Value: "Empty response",
				}
			}
			resp.Result = result.Result

			logger.Debugf("%s (%v)\n", abbreviate(result.Result.Value, 64), len(result.Result.Value))
			if logger.IsTrace() {
				logger.Debugf("result: %s\n", result.Result.Value)
			}

			return next.Serve(req, resp)
		})
	}
}
