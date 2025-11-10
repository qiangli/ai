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

func (r *AIKit) Call(ctx context.Context, vars *api.Vars, owner string, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return atm.CallKit(r, tf.Kit, tf.Name, callArgs...)
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
				var prompt = ""
				if v.Instruction != nil {
					prompt = clip(v.Instruction.Content, 1000)
				}
				return fmt.Sprintf(tpl, v.Name, v.Display, v.Description, prompt), nil
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
	var input *api.UserInput
	if r.h.agent != nil && r.h.agent.RawInput != nil {
		input = r.h.agent.RawInput.Clone()
	} else {
		input = &api.UserInput{}
	}
	req := api.NewRequest(ctx, agent, input)
	req.Parent = r.h.agent

	resp := &api.Response{}

	if err := r.h.sw.execAgent(r.h.agent, req, resp); err != nil {
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
		log.GetLogger(ctx).Debugf("Recalled %v messages in memory less than %v minutes old\n", count, maxSpan)
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

func (r *AIKit) ContextGetMessages(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
	var messages = vars.ListHistory()
	b, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: string(b),
	}, nil
}

func (r *AIKit) ContextSetMessages(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	data, err := atm.GetStrProp("messages", args)
	if err != nil {
		return nil, err
	}

	var messages []*api.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, err
	}
	vars.AddHistory(messages)
	return &api.Result{
		Value: "success",
	}, nil
}

func (r *AIKit) GetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	keys, err := atm.GetArrayProp("keys", args)
	if err != nil {
		return nil, err
	}

	envs := vars.Global.GetEnvs(keys)
	b, err := json.Marshal(envs)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: string(b),
	}, nil
}

func (r *AIKit) SetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	vars.Global.SetEnvs(args)
	return &api.Result{
		Value: "success",
	}, nil
}

func (r *AIKit) UnsetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	keys, err := atm.GetArrayProp("keys", args)
	if err != nil {
		return nil, err
	}

	vars.Global.UnsetEnvs(keys)
	return &api.Result{
		Value: "success",
	}, nil
}

func (r *AIKit) AgentGetPrompt(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	var p string
	if r.h.agent.Instruction != nil {
		p = r.h.agent.Instruction.Content
	}
	return &api.Result{
		Value: p,
	}, nil
}

func (r *AIKit) AgentSetPrompt(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	instruction, err := atm.GetStrProp("instruction", args)
	if err != nil {
		return nil, err
	}

	r.h.agent.Instruction = &api.Instruction{
		Content: instruction,
	}
	return &api.Result{
		Value: "success",
	}, nil
}

func (r *AIKit) AgentGetQuery(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	var q string
	if r.h.agent.RawInput != nil {
		q = r.h.agent.RawInput.Query()
	}
	return &api.Result{
		Value: q,
	}, nil
}

func (r *AIKit) AgentSetQuery(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	query, err := atm.GetStrProp("query", args)
	if err != nil {
		return nil, err
	}

	if r.h.agent.RawInput != nil {
		r.h.agent.RawInput.Message = query
	} else {
		r.h.agent.RawInput = &api.UserInput{
			Message: query,
		}
	}
	return &api.Result{
		Value: "success",
	}, nil
}

func (r *AIKit) AgentGetModel(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	if r.h.agent.Model == nil {
		return nil, fmt.Errorf("Model not set for the current agent")

	}
	m := &api.Model{
		Provider: r.h.agent.Model.Provider,
		BaseUrl:  r.h.agent.Model.BaseUrl,
		Model:    r.h.agent.Model.Model,
		ApiKey:   r.h.agent.Model.ApiKey,
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: string(b),
	}, nil
}

func (r *AIKit) AgentSetModel(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}

	provider, err := atm.GetStrProp("provider", args)
	if err != nil {
		return nil, err
	}
	baseURL, err := atm.GetStrProp("base_url", args)
	if err != nil {
		return nil, err
	}
	model, err := atm.GetStrProp("model", args)
	if err != nil {
		return nil, err
	}
	apiKey, err := atm.GetStrProp("api_key", args)
	if err != nil {
		return nil, err
	}
	r.h.agent.Model = &api.Model{
		Provider: provider,
		BaseUrl:  baseURL,
		Model:    model,
		ApiKey:   apiKey,
	}
	return &api.Result{
		Value: "success",
	}, nil
}

func (r *AIKit) AgentGetTools(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	ids, err := atm.GetStrProp("ids", args)
	if err != nil {
		return nil, err
	}
	var tools []*api.ToolFunc
	if len(ids) > 0 {
		var memo = make(map[string]*api.ToolFunc)
		for _, v := range r.h.agent.Tools {
			memo[v.ID()] = v
		}
		for _, id := range ids {
			k := api.KitName(id).ID()
			if v, ok := memo[k]; ok {
				tools = append(tools, v)
			}
		}
	} else {
		tools = r.h.agent.Tools
	}

	b, err := json.Marshal(tools)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: string(b),
	}, nil
}

func (r *AIKit) AgentSetTools(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	if r.h == nil || r.h.agent == nil {
		return nil, fmt.Errorf("No active agent found")
	}
	ids, err := atm.GetArrayProp("ids", args)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("missing tool ids")
	}
	var memo = make(map[string]struct{})
	for _, k := range ids {
		id := api.KitName(k).ID()
		memo[id] = struct{}{}
	}

	// this implementatin is incomplete
	// TODO caching/mcp tools
	list, err := r.sw.Assets.ListToolkit(r.sw.User.Email)
	if err != nil {
		return nil, err
	}

	var tools []*api.ToolFunc
	for kit, tc := range list {
		for _, v := range tc.Tools {
			id := api.KitName(kit + ":" + v.Name).ID()
			if _, ok := memo[id]; !ok {
				continue
			}
			tool := &api.ToolFunc{
				Kit:         tc.Kit,
				Type:        v.Type,
				Name:        v.Name,
				Description: v.Description,
				Parameters:  v.Parameters,
				Body:        v.Body,
				//
				Agent: v.Agent,
				//
				Provider: nvl(v.Provider, tc.Provider),
				BaseUrl:  nvl(v.BaseUrl, tc.BaseUrl),
				ApiKey:   nvl(v.ApiKey, tc.ApiKey),
				//
			}
			tools = append(tools, tool)
		}
	}

	r.h.agent.Tools = tools

	return &api.Result{
		Value: "success",
	}, nil
}
