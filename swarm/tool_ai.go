package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

type AIKit struct {
	sw    *Swarm
	agent *api.Agent
}

func (r *AIKit) run(ctx context.Context, id string, args map[string]any) (any, error) {
	return r.agent.Runner.Run(ctx, id, args)
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

func (r *AIKit) checkAndCreate(ctx context.Context, vars *api.Vars, tf string, args api.ArgMap) (*api.Agent, error) {
	if v, found := args["agent"]; found {
		if a, ok := v.(*api.Agent); ok {
			return a, nil
		}
		if _, ok := v.(string); ok {
			a, err := r.createAgent(ctx, vars, tf, args)
			if err != nil {
				return nil, err
			}
			return a, nil
		}
	}
	return nil, fmt.Errorf("missing agent")
}

func (r *AIKit) llmAdapter(agent *api.Agent, args api.ArgMap) (api.LLMAdapter, error) {
	var llmAdapter api.LLMAdapter
	if v, found := args["adapter"]; found {
		switch vt := v.(type) {
		case api.LLMAdapter:
			llmAdapter = vt
		case string:
			if v, err := r.sw.Adapters.Get(vt); err != nil {
				return nil, err
			} else {
				llmAdapter = v
			}
		default:
			return nil, fmt.Errorf("adapter not valid: %v", v)
		}
	} else {
		if agent != nil && agent.Adapter != "" {
			if v, err := r.sw.Adapters.Get(agent.Adapter); err == nil {
				llmAdapter = v
			}
		}
	}
	if llmAdapter == nil {
		llmAdapter = &adapter.ChatAdapter{}
	}
	return llmAdapter, nil
}

func (r *AIKit) CallLlm(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (*api.Result, error) {
	var owner = r.sw.User.Email

	var agent = r.agent
	if v, err := r.checkAndCreate(ctx, vars, tf, args); err == nil {
		agent = v
	}

	// query is required
	query, _ := api.GetStrProp("query", args)
	// convert message/content into query (if they exist and are not templates)
	if query == "" {
		query, _ = vars.RTE.DefaultQuery(args)
	}
	// enforce if required is not enforced in args.
	if query == "" {
		return nil, fmt.Errorf("query is required:")
	}

	prompt, _ := api.GetStrProp("prompt", args)

	// model
	var model *api.Model
	if v, found := args["model"]; found {
		switch vt := v.(type) {
		case *api.Model:
			model = vt
		case string:
			// set/level
			set, level := api.Setlevel(vt).Decode()
			v, err := conf.LoadModel(owner, set, level, r.sw.Assets)
			if err != nil {
				return nil, err
			}
			model = v
		}
	}

	if model == nil {
		model = agent.Model
		// return nil, fmt.Errorf("model is required")
	}
	var apiKey = model.ApiKey
	if apiKey == "" {
		// default key
		apiKey = model.Provider
	}

	ak, err := r.sw.Secrets.Get(owner, apiKey)
	if err != nil {
		return nil, err
	}
	token := func() string {
		return ak
	}

	// tools
	var tools []*api.ToolFunc
	if v, found := args["tools"]; found {
		if tf, ok := v.([]*api.ToolFunc); ok {
			tools = tf
		} else {
			var sa []string
			switch vt := v.(type) {
			case []string:
				sa = vt
			case string:
				if err := json.Unmarshal([]byte(vt), &sa); err != nil {
					return nil, err
				}
			}
			if len(sa) > 0 {
				if v, err := r.getTools(sa); err != nil {
					return nil, err
				} else {
					tools = v
				}
			}
		}
	} else {
		tools = agent.Tools
	}

	var req = &api.Request{
		Agent: agent,
	}

	var id = uuid.NewString()
	var history []*api.Message

	// 1. New System Message
	// instruction/system role prompt as first message
	if prompt != "" {
		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: prompt,
			Sender:  "",
		})
	}

	// 2. Context Messages
	// context/history, skip system role
	var messages = api.ToMessages(args["history"])
	for _, msg := range messages {
		if msg.Role != api.RoleSystem {
			history = append(history, msg)
		}
	}

	// 3. New User Message
	// Additional user message/query
	history = append(history, &api.Message{
		ID:      uuid.NewString(),
		Session: id,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: query,
		Sender:  owner,
	})

	// request
	req.Query = query
	req.Prompt = prompt
	req.Messages = history
	req.Arguments = args
	req.Model = model
	req.Token = token
	req.Tools = tools
	req.Runner = agent.Runner

	llmAdapter, err := r.llmAdapter(agent, args)
	if err != nil {
		return nil, err
	}

	resp, err := llmAdapter.Call(ctx, req)

	// update ouput
	args["query"] = query
	args["prompt"] = prompt
	args["history"] = history
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

	list, count, err := listAgents(r.sw.Assets, user)
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
	if agent == "self" {
		if r.agent == nil {
			return "", fmt.Errorf("Sorry, something went terribaly wrong")
		}
		agent = r.agent.Name
	}
	pack, _ := api.Packname(agent).Decode()
	ac, err := r.sw.Assets.FindAgent(r.sw.User.Email, pack)
	if err != nil {
		return "", err
	}

	if ac == nil {
		return "", fmt.Errorf("no config found for %s", agent)
	}
	for _, v := range ac.Agents {
		if api.Packname(v.Name).Equal(agent) {
			var prompt = ""
			if v.Instruction != "" {
				prompt = clip(v.Instruction, 1000)
			}
			return fmt.Sprintf(tpl, v.Name, v.Display, v.Description, prompt), nil
		}
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

func (r *AIKit) ReadAgentConfig(ctx context.Context, vars *api.Vars, _ string, args api.ArgMap) (*api.AppConfig, error) {
	// agent:name -> agent
	// --agent agent
	name, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	if name == "self" {
		if r.agent == nil {
			return nil, fmt.Errorf("Sorry, something went terribaly wrong")
		}
		name = r.agent.Name
	}
	args["kit"] = "agent"
	args["name"] = name

	// cfg, found := args["config"]
	// if found {
	// 	if v, ok := cfg.(*api.AppConfig); ok {
	// 		return v, nil
	// 	}
	// }

	var loader = NewConfigLoader(r.sw.Vars.RTE)

	// cfg, found := args["config"]
	// if found {
	// 	data := api.ToString(cfg)
	// 	if data != "" {
	// 		// load content from config into the data buffer
	// 		if err := loader.LoadContent(data); err != nil {
	// 			return nil, err
	// 		}
	// 	}
	// }

	// load from script
	// TODO only load mime-type yaml
	if v := args.GetString("script"); v != "" {
		if strings.HasSuffix(v, ".yaml") || strings.HasSuffix(v, ".yml") {
			if err := loader.LoadContent(v); err != nil {
				// return nil, err
			}
		}
	}

	config, err := loader.LoadAgentConfig(api.Packname(name))
	if err != nil {
		return nil, err
	}
	// args["config"] = config
	return config, nil
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

func (r *AIKit) SpawnAgent(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	v, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	if v == "" {
		return nil, fmt.Errorf("missing agent name")
	}

	kit := atm.NewSystemKit()
	args["flow_type"] = api.FlowTypeSequence
	args["actions"] = []string{"ai:new_agent", "ai:build_model", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm"}
	result, err := kit.Flow(ctx, vars, "", args)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AIKit) kitname(args map[string]any) api.Kitname {
	am := api.ArgMap(args)
	kit := am.GetString("kit")
	name := am.GetString("name")
	kn := api.Kitname(kit + "/" + name)
	return kn
}

func (r *AIKit) NewAgent(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (*api.Result, error) {
	v, err := r.createAgent(ctx, vars, tf, args)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: fmt.Sprintf("agent %s created", v.Name),
	}, nil
}

func (r *AIKit) ReloadAgent(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	script, err := api.GetStrProp("script", args)
	if err != nil {
		return nil, err
	}
	if script == "" {
		return nil, fmt.Errorf("missing agent configuraiton script file")
	}

	return &api.Result{
		NextAgent: "self",
		Value:     script,
		State:     api.StateTransfer,
	}, nil
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

	list, count, err := listTools(r.sw.Assets, user)
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

func (r *AIKit) ReadToolConfig(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (any, error) {
	tid, err := api.GetStrProp("tool", args)
	if err != nil {
		return "", err
	}

	cfg, found := args["config"]
	if found {
		if v, ok := cfg.(*api.AppConfig); ok {
			return v, nil
		}
	}

	var loader = NewConfigLoader(r.sw.Vars.RTE)
	data := api.ToString(cfg)
	if data != "" {
		loader.LoadContent(data)
	}

	config, err := loader.LoadToolConfig(api.Kitname(tid))
	if err != nil {
		return nil, err
	}
	args["config"] = config
	return config, nil
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
		return r.run(ctx, tid, params)
	}

	// LLM (openai) sometimes does not provide parameters in the args as defined in the tool yaml.
	// returning the error does force to correct this but with multiple calls.
	// we try args instead. if successful, it means correct parameters are provided at the top level.
	log.GetLogger(ctx).Debugf("Tool execute: try ***args*** instead. tid: %s params: %+v\n", tid, args)
	out, err := r.run(ctx, tid, args)
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

	list, count, err := listModels(r.sw.Assets, user)
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
		maxHistory = 7
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

func listAgents(assets api.AssetManager, user string) (string, int, error) {
	agents, err := assets.ListAgent(user)
	if err != nil {
		return "", 0, err
	}

	dict := make(map[string]*api.AgentConfig)
	for _, v := range agents {
		for _, sub := range v.Agents {
			dict[sub.Name] = sub
		}
	}

	keys := make([]string, 0)
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(fmt.Sprintf("%s:\n    %s\n\n", k, dict[k].Description))
	}
	return buf.String(), len(keys), nil
}

func listTools(assets api.AssetManager, user string) (string, int, error) {
	tools, err := assets.ListToolkit(user)
	if err != nil {
		return "", 0, err
	}

	list := []string{}
	for kit, tc := range tools {
		for _, v := range tc.Tools {
			// NOTE: Type in the output seems to confuse LLM (openai)
			list = append(list, fmt.Sprintf("%s:%s: %s\n", kit, v.Name, v.Description))
		}
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), len(list), nil
}

func listModels(assets api.AssetManager, user string) (string, int, error) {
	models, _ := assets.ListModels(user)

	list := []string{}
	for set, tc := range models {
		var keys []string
		for k := range tc.Models {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, level := range keys {
			v := tc.Models[level]
			list = append(list, fmt.Sprintf("%s/%s:\n    %s\n    %s\n    %s\n    %s\n", set, level, v.Provider, v.Model, v.BaseUrl, v.ApiKey))
		}
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), len(list), nil
}
