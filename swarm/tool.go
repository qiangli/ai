package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/log"
)

func callAgentTool(ctx context.Context, sw *Swarm, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	req := api.NewRequest(ctx, tf.Agent, agent.RawInput.Clone())
	req.RawInput.Message, _ = atm.GetStrProp("query", args)
	req.Arguments = args

	resp := &api.Response{}

	if err := sw.Run(req, resp); err != nil {
		return nil, err
	}

	if len(resp.Messages) == 0 {
		return nil, fmt.Errorf("empty result")
	}

	result := &api.Result{
		Value: resp.Messages[len(resp.Messages)-1].Content,
	}

	return result, nil
}

func NewToolCaller(sw *Swarm) api.ToolCaller {
	save := func(v *api.Result) (string, error) {
		id := NewBlobID()
		b := &api.Blob{
			ID:       id,
			MimeType: v.MimeType,
			Content:  []byte(v.Value),
		}
		err := sw.Blobs.Put(id, b)
		return id, err
	}
	toResult := func(v any) *api.Result {
		if r, ok := v.(*api.Result); ok {
			if r.MimeType != api.ContentTypeB64JSON {
				return r
			}
			// image
			// transform media response into data uri
			id, err := save(r)
			if err != nil {
				return &api.Result{
					Value: err.Error(),
				}
			}
			dataURI := fmt.Sprintf("data:application/x.dhnt.blob;mime=%s;%s", r.MimeType, id)
			return &api.Result{
				Value: fmt.Sprintf("The image is available as: %s", dataURI),
			}
		}
		if s, ok := v.(string); ok {
			return &api.Result{
				Value: s,
			}
		}
		return &api.Result{
			Value: fmt.Sprintf("%v", v),
		}
	}

	dispatch := func(ctx context.Context, vars *api.Vars, agent *api.Agent, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
		// agent tool
		if v.Type == api.ToolTypeAgent {
			out, err := callAgentTool(ctx, sw, agent, v, args)
			if err != nil {
				return nil, err
			}
			return toResult(out), nil
		}

		kit, err := sw.Tools.GetKit(v.Type)
		if err != nil {
			return nil, err
		}
		token := func() (string, error) {
			return sw.Secrets.Get(agent.Owner, v.ApiKey)
		}
		//
		out, err := kit.Call(ctx, vars, token, v, args)
		if err != nil {
			return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
		}
		return toResult(out), nil
	}

	return func(vars *api.Vars, agent *api.Agent) func(context.Context, string, map[string]any) (*api.Result, error) {
		toolMap := make(map[string]*api.ToolFunc)
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}

		return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
			v, ok := toolMap[tid]
			if !ok {
				return nil, fmt.Errorf("tool not found: %s", tid)
			}

			log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", v.Kit, v.Name, args)

			// add model to system kit for command evaluation
			if v.Type == api.ToolTypeSystem {
				ctx = context.WithValue(ctx, atm.ModelsContextKey, agent.Model)
			}

			result, err := dispatch(ctx, vars, agent, v, args)

			if err != nil {
				log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
			} else {
				log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
			}

			return result, err
		}
	}
}
