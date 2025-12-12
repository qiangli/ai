package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

type AIKit struct {
	sw    *Swarm
	agent *api.Agent
}

func (r *AIKit) Run(ctx context.Context, id string, args map[string]any) (any, error) {
	return NewAgentToolRunner(r.sw, r.sw.User.Email, r.agent).Run(ctx, id, args)
}

func NewAIKit(sw *Swarm, agent *api.Agent) *AIKit {
	return &AIKit{
		sw:    sw,
		agent: agent,
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

func (r *AIKit) Call(ctx context.Context, vars *api.Vars, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	return atm.CallKit(r, tf.Kit, tf.Name, callArgs...)
}

func (r *AIKit) CallLlm(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	query, err := api.GetStrProp("query", args)
	if err != nil {
		return nil, err
	}
	if query == "" {
		return nil, fmt.Errorf("missing query")
	}
	prompt, _ := api.GetStrProp("prompt", args)
	provider, _ := api.GetStrProp("provider", args)
	if provider == "" {
		provider = "openai"
	}
	arguments, _ := structToMap(args["arguments"])
	tools, _ := api.GetArrayProp("tools", args)

	owner := r.sw.User.Email

	var req = &llm.Request{}
	var messages []*api.Message

	id := uuid.NewString()
	if prompt != "" {
		messages = append(messages, &api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: prompt,
			Sender:  "",
		})
	}
	messages = append(messages, &api.Message{
		ID:      uuid.NewString(),
		Session: id,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: query,
		Sender:  owner,
	})
	req.Messages = messages

	// model set: provider
	// model, err := conf.LoadModel(owner, provider, "any", r.sw.Assets)
	model, err := r.getModel(provider)
	if err != nil {
		return nil, err
	}
	ak, err := r.sw.Secrets.Get(owner, provider)
	if err != nil {
		return nil, err
	}
	token := func() string {
		return ak
	}
	req.Model = model
	req.Token = token

	// tools
	if len(tools) > 0 {
		if v, err := r.getTools(tools); err != nil {
			return nil, err
		} else {
			req.Tools = v
		}
	}

	// LLM parameters
	if arguments != nil {
		req.Arguments = api.NewArguments().AddArgs(arguments)
	}

	// var adapter = &adapter.ResponseAdapter{}
	var adapter = &adapter.ChatAdapter{}
	resp, err := adapter.Call(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Result == nil {
		resp.Result = &api.Result{
			Value: "Empty response",
		}
	}
	return resp.Result, nil
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

func (r *AIKit) GetAgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	const tpl = `
Agent: %s
Display: %s
Description: %s
Instruction: %s
`

	agent, err := api.GetStrProp("agent", args)
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
				if v.Instruction != "" {
					prompt = clip(v.Instruction, 1000)
				}
				return fmt.Sprintf(tpl, v.Name, v.Display, v.Description, prompt), nil
			}
		}
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

func (r *AIKit) TransferAgent(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}

	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}

func (r *AIKit) SpawnAgent(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	name, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("missing agent name")
	}

	return r.sw.runm(ctx, r.agent, name, args)
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

func (r *AIKit) GetToolInfo(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	const tpl = `
Tool: %s__%s
Description: %s
Parameters: %s
`
	log.GetLogger(ctx).Debugf("Tool info: %s %+v\n", tf, args)

	tid, err := api.GetStrProp("tool", args)
	if err != nil {
		return "", err
	}

	kit, name := api.Kitname(tid).Decode()

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

func (r *AIKit) ExecuteTool(ctx context.Context, _ *api.Vars, tf string, args map[string]any) (any, error) {
	log.GetLogger(ctx).Debugf("Tool execute: %s %+v\n", tf, args)

	tid, err := api.GetStrProp("tool", args)
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
		return r.Run(ctx, tid, params)
	}

	// LLM (openai) sometimes does not provide parameters in the args as defined in the tool yaml.
	// returning the error does force to correct this but with multiple calls.
	// we try args instead. if successful, it means correct parameters are provided at the top level.
	log.GetLogger(ctx).Debugf("Tool execute: try ***args*** instead. tid: %s params: %+v\n", tid, args)
	out, err := r.Run(ctx, tid, args)
	if err != nil {
		return nil, fmt.Errorf("required parameters not found in: %+v. error: %v", args, err)
	}
	return out, nil
}

func (r *AIKit) GetToolCalllog(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("Tool call log: %s %+v\n", tf, args)
	v, err := vars.ToolCalllog()
	return v, err
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

	maxHistory, err := api.GetIntProp("max_history", args)
	if err != nil || maxHistory <= 0 {
		maxHistory = 3
	}
	maxSpan, err := api.GetIntProp("max_span", args)
	if err != nil || maxSpan <= 0 {
		maxSpan = 1440
	}
	offset, err := api.GetIntProp("offset", args)
	if err != nil || offset <= 0 {
		offset = 0
	}
	roles, err := api.GetArrayProp("roles", args)
	if err != nil || len(roles) == 0 {
		roles = []string{"assistant", "user"}
	}

	var option = &api.MemOption{
		MaxHistory: maxHistory,
		MaxSpan:    maxSpan,
		Offset:     offset,
		Roles:      roles,
	}
	list, count, err := conf.ListHistory(r.sw.History, option)
	if err != nil {
		return "", fmt.Errorf("Failed to recall messages (%s): %v", option, err)
	}
	if count > 0 {
		log.GetLogger(ctx).Debugf("Recalled %v messages in memory less than %v minutes old\n", count, maxSpan)
	}

	var v = fmt.Sprintf("Available messages (%s): %v\n\n%s\n", option, count, list)
	return v, nil
}

func (r *AIKit) GetMessageInfo(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	id, err := api.GetStrProp("id", args)
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

// func (r *AIKit) ContextGetMessages(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
// 	// var messages = r.sw.Vars.ListHistory()
// 	messages := r.agent.History()
// 	if len(messages) == 0 {
// 		return &api.Result{}, nil
// 	}
// 	b, err := json.Marshal(messages)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &api.Result{
// 		Value: string(b),
// 	}, nil
// }

// func (r *AIKit) ContextSetMessages(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
// 	data, err := api.GetStrProp("messages", args)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var messages []*api.Message
// 	if err := json.Unmarshal([]byte(data), &messages); err != nil {
// 		return nil, err
// 	}
// 	// r.sw.Vars.SetHistory(messages)
// 	r.agent.SetHistory(messages)
// 	return &api.Result{
// 		Value: "success",
// 	}, nil
// }

func (r *AIKit) GetEnvs(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	keys, err := api.GetArrayProp("keys", args)
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
	keys, err := api.GetArrayProp("keys", args)
	if err != nil {
		return nil, err
	}

	vars.Global.UnsetEnvs(keys)
	return &api.Result{
		Value: "success",
	}, nil
}

// func (r *AIKit) AgentGetPrompt(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	var p string
// 	if r.agent != nil {
// 		p = r.agent.Prompt()
// 	}
// 	return &api.Result{
// 		Value: p,
// 	}, nil
// }

// func (r *AIKit) AgentSetPrompt(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	instruction, err := api.GetStrProp("instruction", args)
// 	if err != nil {
// 		return nil, err
// 	}

// 	r.agent.SetPrompt(instruction)
// 	return &api.Result{
// 		Value: "success",
// 	}, nil
// }

// func (r *AIKit) AgentGetQuery(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	return &api.Result{
// 		Value: r.agent.Query(),
// 	}, nil
// }

// func (r *AIKit) AgentSetQuery(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	query, err := api.GetStrProp("query", args)
// 	if err != nil {
// 		return nil, err
// 	}

// 	r.agent.SetQuery(query)
// 	return &api.Result{
// 		Value: "success",
// 	}, nil
// }

// func (r *AIKit) AgentGetModel(_ context.Context, vars *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	if r.agent.Model == nil {
// 		return nil, fmt.Errorf("Model not set for the current agent")

// 	}
// 	m := &api.Model{
// 		Provider: r.agent.Model.Provider,
// 		BaseUrl:  r.agent.Model.BaseUrl,
// 		Model:    r.agent.Model.Model,
// 		ApiKey:   r.agent.Model.ApiKey,
// 	}
// 	b, err := json.Marshal(m)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &api.Result{
// 		Value: string(b),
// 	}, nil
// }

// func (r *AIKit) AgentSetModel(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}

// 	provider, err := api.GetStrProp("provider", args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	baseURL, err := api.GetStrProp("base_url", args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	model, err := api.GetStrProp("model", args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	apiKey, err := api.GetStrProp("api_key", args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	r.agent.Model = &api.Model{
// 		Provider: provider,
// 		BaseUrl:  baseURL,
// 		Model:    model,
// 		ApiKey:   apiKey,
// 	}
// 	return &api.Result{
// 		Value: "success",
// 	}, nil
// }

// func (r *AIKit) AgentGetTools(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	ids, err := api.GetStrProp("ids", args)
// 	if err != nil {
// 		return nil, err
// 	}
// 	var tools []*api.ToolFunc
// 	if len(ids) > 0 {
// 		var memo = make(map[string]*api.ToolFunc)
// 		for _, v := range r.agent.Tools {
// 			memo[v.ID()] = v
// 		}
// 		for _, id := range ids {
// 			k := api.Kitname(id).ID()
// 			if v, ok := memo[k]; ok {
// 				tools = append(tools, v)
// 			}
// 		}
// 	} else {
// 		tools = r.agent.Tools
// 	}

// 	b, err := json.Marshal(tools)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &api.Result{
// 		Value: string(b),
// 	}, nil
// }

// func (r *AIKit) AgentSetTools(_ context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
// 	if r.agent == nil {
// 		return nil, fmt.Errorf("No active agent found")
// 	}
// 	ids, err := api.GetArrayProp("ids", args)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if len(ids) == 0 {
// 		return nil, fmt.Errorf("missing tool ids")
// 	}

// 	tools, err := r.getTools(ids)
// 	if err != nil {
// 		return nil, err
// 	}
// 	r.agent.Tools = tools

// 	return &api.Result{
// 		Value: "success",
// 	}, nil
// }

// return tools by tool kit:name or ids
func (r *AIKit) getTools(ids []string) ([]*api.ToolFunc, error) {
	var memo = make(map[string]struct{})
	for _, k := range ids {
		id := api.Kitname(k).ID()
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
			id := api.Kitname(kit + ":" + v.Name).ID()
			if _, ok := memo[id]; !ok {
				continue
			}
			tool := &api.ToolFunc{
				Kit:         tc.Kit,
				Type:        api.ToolType(v.Type),
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
	return tools, nil
}

// return built-in model
func (r *AIKit) getModel(provider string) (*api.Model, error) {
	m, ok := conf.DefaultModels[provider]
	if !ok {
		return nil, fmt.Errorf("model not found for provider: %s", provider)
	}
	return m, nil
}
