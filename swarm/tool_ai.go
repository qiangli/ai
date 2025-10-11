package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

type AIKit struct {
	sw       *Swarm
	h        *agentHandler
	callTool api.ToolRunner
}

func NewAIKit(h *agentHandler) *AIKit {
	return &AIKit{
		sw:       h.sw,
		h:        h,
		callTool: h.createAICaller(),
	}
}

type ListCacheKey struct {
	Type string
	User string
}

var (
	listAgentsCache = expirable.NewLRU[ListCacheKey, string](10000, nil, time.Second*900)
	listToolsCache  = expirable.NewLRU[ListCacheKey, string](10000, nil, time.Second*900)
)

func (r *AIKit) Call(ctx context.Context, vars *api.Vars, _ api.SecretToken, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return atm.CallKit(r, tf.Config.Kit, tf.Name, callArgs...)
}

func (r *AIKit) ListAgents(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List agents: %s %+v\n", tf, args)

	var user = r.sw.User.Email
	// cached list
	key := ListCacheKey{
		Type: "agent",
		User: user,
	}
	if v, ok := listAgentsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached agent list: %+v\n", key)
		return v, nil
	}

	list, count, err := conf.ListAgents(r.sw.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available agents: %v\n\n%s\n", count, list)
	listAgentsCache.Add(key, v)

	return v, nil
}

func (r *AIKit) AgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	const tpl = `
Agent: %s
Display: %s
Description: %s
Instruction: %s
`

	agent, err := atm.GetStrProp("agent", args)
	if err != nil {
		return "", err
	}
	ac, err := r.sw.Assets.FindAgent(r.sw.User.Email, agent)
	if err != nil {
		return "", err
	}

	if ac != nil {
		for _, v := range ac.Agents {
			if v.Name == agent {
				return fmt.Sprintf(tpl, v.Name, v.Display, v.Description, clip(v.Instruction.Content, 1000)), nil
			}
		}
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

func (r *AIKit) AgentTransfer(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := atm.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}

	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}

func (r *AIKit) AgentSpawn(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := atm.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}

	// prevent loop
	if r.h.agent.Name == agent {
		return nil, fmt.Errorf("you are %q! Calling yourself not permitted", agent)
	}

	req := api.NewRequest(ctx, agent, r.h.agent.RawInput.Clone())
	resp := &api.Response{}

	if err := r.h.exec(req, resp); err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (r *AIKit) ListTools(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List tools: %s %+v\n", tf, args)

	var user = r.sw.User.Email
	// cached list
	key := ListCacheKey{
		Type: "tool",
		User: user,
	}
	if v, ok := listToolsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached tool list: %+v\n", key)
		return v, nil
	}

	list, count, err := conf.ListTools(r.sw.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available tools: %v\n\n%s\n", count, list)
	listToolsCache.Add(key, v)

	return v, nil
}

func (r *AIKit) ToolInfo(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	const tpl = `
Tool: %s__%s
Description: %s
Parameters: %s
`
	log.GetLogger(ctx).Debugf("Tool info: %s %+v\n", tf, args)

	tid, err := atm.GetStrProp("tool", args)
	if err != nil {
		return "", err
	}

	kit, name := api.KitName(tid).Decode()

	tc, err := r.sw.Assets.FindToolkit(r.sw.User.Email, kit)
	if err != nil {
		return "", err
	}

	if tc != nil {
		for _, v := range tc.Tools {
			if v.Name == name {
				params, err := json.Marshal(v.Parameters)
				if err != nil {
					return "", err
				}
				// TODO params may need better handling
				log.GetLogger(ctx).Debugf("Tool info: %s %+v\n", tid, string(params))
				return fmt.Sprintf(tpl, kit, v.Name, v.Description, string(params)), nil
			}
		}
	}
	return "", fmt.Errorf("unknown tool: %s", tid)
}

func (r *AIKit) ToolExecute(ctx context.Context, _ *api.Vars, tf string, args map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Debugf("Tool execute: %s %+v\n", tf, args)

	tid, err := atm.GetStrProp("tool", args)
	if err != nil {
		return nil, err
	}

	v, ok := args["parameters"]
	if ok {
		params, err := structToMap(v)
		if err != nil {
			return nil, err
		}
		log.GetLogger(ctx).Debugf("Tool execute: tid: %s params: %+v\n", tid, params)
		return r.callTool(ctx, tid, params)
	}

	// LLM (openai) sometimes does not provide parameters in the args as defined in the tool yaml.
	// returning the error does force to correct this but with multiple calls.
	// we try args instead. if successful, it means correct parameters are provided at the top level.
	log.GetLogger(ctx).Debugf("Tool execute: try ***args*** instead. tid: %s params: %+v\n", tid, args)
	out, err := r.callTool(ctx, tid, args)
	if err != nil {
		return nil, fmt.Errorf("required parameters not found in: %+v. error: %v", args, err)
	}
	return out, nil
}

func (r *AIKit) ListModels(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List models: %s %+v\n", tf, args)

	var user = r.sw.User.Email
	// cached list
	key := ListCacheKey{
		Type: "model",
		User: user,
	}
	if v, ok := listToolsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached model list: %+v\n", key)
		return v, nil
	}

	list, count, err := conf.ListModels(r.sw.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available models: %v\n\n%s\n", count, list)
	listToolsCache.Add(key, v)

	return v, nil
}

func (r *AIKit) ListMessages(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List messages: %s %+v\n", tf, args)

	maxHistory, err := atm.GetIntProp("max_history", args)
	if err != nil {
		return "", err
	}
	maxSpan, err := atm.GetIntProp("max_span", args)
	if err != nil {
		return "", err
	}

	list, count, err := conf.ListHistory(r.sw.History, &api.MemOption{
		MaxHistory: maxHistory,
		MaxSpan:    maxSpan,
	})
	if err != nil {
		return "", err
	}
	if count > 0 {
		log.GetLogger(ctx).Infof("⣿ recalling %v messages in memory less than %v minutes old\n", count, maxSpan)
	}

	var v = fmt.Sprintf("Available messages: %v\n\n%s\n", count, list)
	return v, nil
}

func (r *AIKit) MessageInfo(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	id, err := atm.GetStrProp("id", args)
	if err != nil {
		return nil, err
	}

	v, err := r.sw.History.Get(id)
	if err != nil {
		return nil, err
	}

	b, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: string(b),
	}, nil
}
