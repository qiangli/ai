package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

func (h *agentHandler) createCaller() api.ToolRunner {
	toolMap := make(map[string]*api.ToolFunc)
	for _, v := range h.agent.Tools {
		toolMap[v.ID()] = v
	}

	return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
		v, ok := toolMap[tid]
		if !ok {
			return nil, fmt.Errorf("tool not found: %s", tid)
		}
		return h.callTool(ctx, v, args)
	}
}

func (h *agentHandler) createAICaller() api.ToolRunner {
	return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
		tools, err := conf.LoadToolFunc(h.agent.Owner, tid, h.sw.Secrets, h.sw.Assets)
		if err != nil {
			return nil, err
		}
		for _, v := range tools {
			id := v.ID()
			if id == tid {
				return h.callTool(ctx, v, args)
			}
		}
		return nil, fmt.Errorf("invalid tool: %s", tid)
	}
}

func (h *agentHandler) callTool(ctx context.Context, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", tf.Kit, tf.Name, args)

	// add model to system kit for command evaluation
	if tf.Type == api.ToolTypeSystem {
		ctx = context.WithValue(ctx, atm.ModelsContextKey, h.agent.Model)
	}

	// result, err := h.dispatch(ctx, h.vars, h.agent, v, args)
	result, err := h.dispatch(ctx, tf, args)

	if err != nil {
		log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
	} else {
		log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
	}

	return result, err
}

// sw *Swarm, agent *api.Agent,
func (h *agentHandler) callAgentTool(ctx context.Context, tf *api.ToolFunc, args map[string]any) (any, error) {
	req := api.NewRequest(ctx, tf.Agent, h.agent.RawInput.Clone())
	req.RawInput.Message, _ = atm.GetStrProp("query", args)
	req.Arguments = args

	resp := &api.Response{}

	if err := h.sw.Run(req, resp); err != nil {
		return nil, err
	}

	if resp.Result == nil {
		return nil, fmt.Errorf("empty result")
	}

	return resp.Result, nil
}

// vars *api.Vars, agent *api.Agent,
func (h *agentHandler) dispatch(ctx context.Context, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// agent tool
	if v.Type == api.ToolTypeAgent {
		// out, err := h.callAgentTool(ctx, h.sw, agent, v, args)
		out, err := h.callAgentTool(ctx, v, args)
		if err != nil {
			return nil, err
		}
		return h.toResult(out), nil
	}

	// TODO a new tool type?
	if v.Type == api.ToolTypeFunc && v.Kit == "ai" {
		aiKit := &AIKit{
			user:     h.sw.User,
			assets:   h.sw.Assets,
			callTool: h.createAICaller(),
		}

		out, err := aiKit.Call(ctx, h.vars, nil, v, args)
		if err != nil {
			return nil, err
		}
		return h.toResult(out), nil
	}

	//
	kit, err := h.sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}
	token := func() (string, error) {
		return h.sw.Secrets.Get(h.agent.Owner, v.ApiKey)
	}

	out, err := kit.Call(ctx, h.vars, token, v, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
	}
	return h.toResult(out), nil
}

func (h *agentHandler) toResult(v any) *api.Result {
	if r, ok := v.(*api.Result); ok {
		if r.MimeType != api.ContentTypeB64JSON {
			return r
		}
		// image
		// transform media response into data uri
		id, err := h.save(r)
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

func (h *agentHandler) save(v *api.Result) (string, error) {
	id := NewBlobID()
	b := &api.Blob{
		ID:       id,
		MimeType: v.MimeType,
		Content:  v.Content,
	}
	err := h.sw.Blobs.Put(id, b)
	return id, err
}
