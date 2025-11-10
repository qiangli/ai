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

func (sw *Swarm) buildAgentToolMap(agent *api.Agent) map[string]*api.ToolFunc {
	toolMap := make(map[string]*api.ToolFunc)
	// inherit tools of embedded agents
	for _, agent := range agent.Embed {
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}
	}
	// the active agent
	for _, v := range agent.Tools {
		toolMap[v.ID()] = v
	}
	return toolMap
}

func (sw *Swarm) createCaller(user *api.User, agent *api.Agent) api.ToolRunner {
	toolMap := sw.buildAgentToolMap(agent)

	return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
		v, ok := toolMap[tid]
		if !ok {
			return nil, fmt.Errorf("tool not found: %s", tid)
		}

		return sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, user), agent, v, args)
	}
}

func (sw *Swarm) createAICaller(agent *api.Agent) api.ToolRunner {
	return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
		tools, err := conf.LoadToolFunc(agent.Owner, tid, sw.Secrets, sw.Assets)
		if err != nil {
			return nil, err
		}
		for _, v := range tools {
			id := v.ID()
			if id == tid {
				return sw.callTool(ctx, agent, v, args)
			}
		}
		return nil, fmt.Errorf("invalid tool: %s", tid)
	}
}

func (sw *Swarm) callTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", tf.Kit, tf.Name, formatArgs(args))

	// add model to system kit for command evaluation
	if tf.Type == api.ToolTypeSystem {
		ctx = context.WithValue(ctx, atm.ModelsContextKey, agent.Model)
	}

	//
	result, err := sw.dispatch(ctx, agent, tf, args)

	if err != nil {
		log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
	} else {
		log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
	}

	return result, err
}

func (sw *Swarm) callAgentType(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// agent tool
	if tf.Kit == api.ToolTypeAgent {
		return sw.callAgentTool(ctx, agent, tf, args)
	}

	// ai tool
	if tf.Kit == "ai" {
		return sw.callAITool(ctx, agent, tf, args)
	}
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (sw *Swarm) callAgentTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// NOTE: is original input always appropriate for the tools?
	req := api.NewRequest(ctx, tf.Agent, agent.RawInput.Clone())
	req.Parent = agent
	req.RawInput.Message, _ = atm.GetStrProp("query", args)
	req.Arguments = args

	resp := &api.Response{}

	err := sw.RunSub(agent, req, resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (sw *Swarm) callAITool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	aiKit := NewAIKit(sw, agent)
	return aiKit.Call(ctx, sw.Vars, "", tf, args)
}

// vars *api.Vars, agent *api.Agent,
func (sw *Swarm) dispatch(ctx context.Context, agent *api.Agent, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// agent tool
	if v.Type == api.ToolTypeAgent {
		out, err := sw.callAgentType(ctx, agent, v, args)
		if err != nil {
			return nil, err
		}
		return sw.toResult(out), nil
	}

	// custom kits
	kit, err := sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}

	env := &api.ToolEnv{
		Owner: agent.Owner,
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
	}
	return sw.toResult(out), nil
}

func (sw *Swarm) toResult(v any) *api.Result {
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
		return &api.Result{
			MimeType: r.MimeType,
			Value:    dataURL(r.MimeType, r.Content),
		}
		// // image
		// // transform media response into data url
		// presigned, err := h.save(r)
		// if err != nil {
		// 	return &api.Result{
		// 		Value: err.Error(),
		// 	}
		// }

		// return &api.Result{
		// 	MimeType: r.MimeType,
		// 	Value:    presigned,
		// }
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

// flow actions
func (sw *Swarm) doAction(ctx context.Context, agent *api.Agent, tf *api.ToolFunc) (*api.Result, error) {
	env := sw.globalEnv()

	var runTool = sw.createCaller(sw.User, agent)
	result, err := runTool(ctx, tf.ID(), env)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("no result")
	}

	// TODO check states?
	sw.Vars.Global.Set(globalResult, result.Value)
	return result, nil
}

// // save and get the presigned url
// func (sw *Swarm) save(v *api.Result) (string, error) {
// 	id := NewBlobID()
// 	b := &api.Blob{
// 		ID:       id,
// 		MimeType: v.MimeType,
// 		Content:  v.Content,
// 	}
// 	err := sw.Blobs.Put(id, b)
// 	if err != nil {
// 		return "", err
// 	}
// 	return sw.Blobs.Presign(id)
// }
