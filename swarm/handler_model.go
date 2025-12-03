package swarm

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

// ModelMiddleware loads the dynamical modle
func ModelMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			ctx := req.Context()
			log.GetLogger(ctx).Debugf("🔗 (model): %s model: %s\n", agent.Name, agent.Model)

			var model *api.Model = agent.Model

			if conf.IsAgentTool(model.Model) {
				out, err := sw.expandx(req.Context(), agent, model.Model)
				if err != nil {
					return err
				}
				var v api.Model
				if err := json.Unmarshal([]byte(out), &v); err != nil {
					return err
				}
				model = &v
			}

			// fill default values based on provider
			// only provider is required
			if model.Provider == "" {
				return fmt.Errorf("model is invalid. %+v", model)
			}
			if model.ApiKey == "" || model.BaseUrl == "" || model.Model == "" {
				m, ok := conf.DefaultModels[model.Provider]
				if !ok {
					return fmt.Errorf("invaid model: %s", model.Provider)
				}
				model.ApiKey = m.ApiKey
				model.BaseUrl = m.BaseUrl
				if model.Model == "" {
					model.Model = m.Model
				}
			}

			//
			ak, err := sw.Secrets.Get(sw.User.Email, model.ApiKey)
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
