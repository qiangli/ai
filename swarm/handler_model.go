package swarm

import (
	"encoding/json"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// ModelMiddleware loads the dynamical modle
func ModelMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			ctx := req.Context()
			log.GetLogger(ctx).Debugf("🔗 (model): %s model: %s\n", agent.Name, agent.Model)

			var model *api.Model = agent.Model

			at, found := parseAgentCommand(model.Model)
			if found {
				out, err := sw.callAgent(agent, req, at.Name, at.Message)
				if err != nil {
					return err
				}
				var v api.Model
				if err := json.Unmarshal([]byte(out), &v); err != nil {
					return err
				}
				model = &v
			}

			//
			ak, err := sw.Secrets.Get(agent.Owner, model.ApiKey)
			if err != nil {
				return err
			}
			token := func() string {
				return ak
			}

			req.Model = model
			req.Token = token

			err = next.Serve(req, resp)

			return err
		})
	}
}
