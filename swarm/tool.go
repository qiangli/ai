package swarm

import (
	"context"
	"fmt"
	"strings"

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

func (h *agentHandler) callAgentType(ctx context.Context, tf *api.ToolFunc, args map[string]any) (any, error) {
	// agent tool
	if tf.Kit == api.ToolTypeAgent {
		return h.callAgentTool(ctx, tf, args)
	}

	// ai tool
	if tf.Kit == "ai" {
		return h.callAITool(ctx, tf, args)
	}
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (h *agentHandler) callAgentTool(ctx context.Context, tf *api.ToolFunc, args map[string]any) (any, error) {
	// NOTE: is original input always appropriate for the tools?
	req := api.NewRequest(ctx, tf.Agent, h.agent.RawInput.Clone())
	req.Parent = h.agent
	req.RawInput.Message, _ = atm.GetStrProp("query", args)
	req.Arguments = args

	resp := &api.Response{}

	err := h.exec(req, resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (h *agentHandler) callAITool(ctx context.Context, tf *api.ToolFunc, args map[string]any) (any, error) {
	aiKit := NewAIKit(h)
	return aiKit.Call(ctx, h.vars, nil, tf, args)
}

// vars *api.Vars, agent *api.Agent,
func (h *agentHandler) dispatch(ctx context.Context, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// agent tool
	if v.Type == api.ToolTypeAgent {
		out, err := h.callAgentType(ctx, v, args)
		if err != nil {
			return nil, err
		}
		return h.toResult(out), nil
	}

	// custom kits
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
		if len(r.Content) == 0 {
			return r
		}
		if r.MimeType == api.ContentTypeImageB64 {
			return r
		}
		if strings.HasPrefix(r.MimeType, "text/") {
			return &api.Result{
				MimeType: r.MimeType,
				Value:    string(r.Content),
			}
		}
		// image
		// transform media response into data uri
		// id, err := h.save(r)
		// if err != nil {
		// 	return &api.Result{
		// 		Value: err.Error(),
		// 	}
		// }
		// dataURI := fmt.Sprintf("data:application/x.dhnt.blob;mime=%s;%s", r.MimeType, id)
		data := dataURL(r.MimeType, r.Content)
		return &api.Result{
			MimeType: r.MimeType,
			Value:    data,
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
