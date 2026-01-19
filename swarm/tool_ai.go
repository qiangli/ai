package swarm

import (
	"bytes"
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
	// sw    *Swarm
	vars *api.Vars
	// agent *api.Agent
}

func NewAIKit(vars *api.Vars) *AIKit {
	return &AIKit{
		vars: vars,
		// agent: agent,
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

func (r *AIKit) Call(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, parent, tf, args}
	return atm.CallKit(r, tf.Kit, tf.Name, callArgs...)
}

func (r *AIKit) checkAndCreate(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (*api.Agent, error) {
	if v, found := args["agent"]; found {
		if a, ok := v.(*api.Agent); ok {
			return a, nil
		}
		if _, ok := v.(string); ok {
			a, err := r.createAgent(ctx, vars, parent, tf, args)
			if err != nil {
				return nil, err
			}
			return a, nil
		}
	}
	return nil, fmt.Errorf("missing agent")
}

func (r *AIKit) llmAdapter(agent *api.Agent, args map[string]any) (api.LLMAdapter, error) {
	// mock if echo__<agent__pack__sub> is found in args.
	if agent != nil && len(args) > 0 {
		id := api.NewPackname(agent.Pack, agent.Name).ID()
		if _, ok := args["echo__"+id]; ok {
			return &adapter.EchoAdapter{}, nil
		}
	}
	//
	var llmAdapter api.LLMAdapter
	if v, found := args["adapter"]; found {
		switch vt := v.(type) {
		case api.LLMAdapter:
			llmAdapter = vt
		case string:
			if v, err := r.vars.Adapters.Get(vt); err != nil {
				return nil, err
			} else {
				llmAdapter = v
			}
		default:
			return nil, fmt.Errorf("adapter not valid: %v", v)
		}
	} else {
		if agent != nil && agent.Adapter != "" {
			if v, err := r.vars.Adapters.Get(agent.Adapter); err == nil {
				llmAdapter = v
			}
		}
	}
	if llmAdapter == nil {
		llmAdapter = &adapter.ChatAdapter{}
		// TODO openai response api tool call issue
		// llmAdapter = &adapter.TextAdapter{}
	}

	return llmAdapter, nil
}

func (r *AIKit) CallLlm(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	var owner = r.vars.User.Email

	var agent = parent
	if v, err := r.checkAndCreate(ctx, vars, parent, tf, args); err == nil {
		agent = v
	}

	// query is required
	query, _ := api.GetStrProp("query", args)
	// convert message/content into query (if they exist and are not templates)
	if query == "" {
		query, _ = vars.DefaultQuery(args)
	}
	// // enforce if required is not enforced in args.
	// if query == "" {
	// 	return nil, fmt.Errorf("query is required for agent: %s/%s", agent.Pack, agent.Name)
	// }

	// prompt is optional
	prompt, _ := api.GetStrProp("prompt", args)
	if prompt == "" {
		prompt, _ = vars.DefaultPrompt(args)
	}

	// resolve model
	// search parents
	// load external
	var model *api.Model
	if v, found := args["model"]; found {
		switch vt := v.(type) {
		case *api.Model:
			model = vt
		case string:
			// set/level
			set, level := api.Setlevel(vt).Decode()
			// embeded/inherited
			if v := findModel(agent, set, level); v != nil {
				model = v
				break
			}
			// external
			v, err := conf.LoadModel(owner, set, level, r.vars.Assets)
			if err != nil {
				return nil, err
			}
			model = v
		}
	}
	// default/any
	if model == nil {
		model = r.vars.RootAgent.Model
	}

	var apiKey = model.ApiKey
	if apiKey == "" {
		// default key
		apiKey = model.Provider
	}

	ak, err := r.vars.Secrets.Get(owner, apiKey)
	if err != nil {
		return nil, err
	}
	token := func() string {
		return ak
	}

	// tools
	funcMap := make(map[string]*api.ToolFunc)
	if v, found := args["tools"]; found {
		var tools []*api.ToolFunc

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
		for _, fn := range tools {
			id := fn.ID()
			if id == "" {
				return nil, fmt.Errorf("tool ID is empty: %v", fn)
			}
			funcMap[id] = fn
		}
	}
	// merge
	for _, fn := range agent.Tools {
		id := fn.ID()
		if id == "" {
			return nil, fmt.Errorf("tool ID is empty: %v", fn)
		}
		funcMap[id] = fn
	}
	var tools []*api.ToolFunc
	for _, v := range funcMap {
		tools = append(tools, v)
	}
	agent.Tools = tools

	// request
	var packname = api.NewPackname(agent.Pack, agent.Name)
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
			Agent:   packname,
		})
	}

	// 2. Context Messages
	// context/history, skip system role and old context message
	var messages = api.ToMessages(args["history"])
	for _, msg := range messages {
		if msg.Role == api.RoleSystem || msg.Context {
			continue
		}
		msg.Context = true
		history = append(history, msg)
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
		Agent:   packname,
	})

	// request
	var req = &api.Request{
		Agent: agent,
	}

	req.Query = query
	req.Prompt = prompt
	req.Messages = history
	req.Arguments = args
	req.Model = model
	req.Token = token
	req.Tools = tools
	req.Runner = agent.Runner

	// ensure defaults
	if req.MaxTurns() == 0 {
		req.SetMaxTurns(adapter.DefaultMaxTurns)
	}

	// call LLM
	llmAdapter, err := r.llmAdapter(agent, args)
	if err != nil {
		return nil, err
	}

	resp, err := llmAdapter.Call(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Result == nil {
		resp.Result = &api.Result{
			Value: "Empty response",
		}
	}

	if resp.Result.State == api.StateTransfer {
		args["agent"] = resp.Result.NextAgent
		return r.SpawnAgent(ctx, vars, parent, tf, args)
	}

	// response message
	message := api.Message{
		ID:      uuid.NewString(),
		Session: id,
		Created: time.Now(),
		//
		ContentType: resp.Result.MimeType,
		Content:     resp.Result.Value,
		//
		Role: nvl(resp.Result.Role, api.RoleAssistant),
		//
		Sender: model.Provider,
		Agent:  packname,
	}
	history = append(history, &message)

	// update ouput
	args["query"] = query
	args["prompt"] = prompt
	args["history"] = history
	args["result"] = resp.Result.Value

	// save the context
	if err := r.vars.History.Save(history); err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (r *AIKit) ListAgents(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List agents: %s %+v\n", tf, args)

	var user = r.vars.User.Email
	// cached list
	key := ListCacheKey{
		Type: "agent",
		User: user,
	}
	if v, ok := listAgentsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached agent list: %+v\n", key)
		return v, nil
	}

	list, count, err := listAgents(r.vars.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available agents: %v\n\n%s\n", count, list)
	listAgentsCache.Add(key, v)

	return v, nil
}

func (r *AIKit) GetAgentInfo(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (string, error) {
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
		if parent == nil {
			return "", fmt.Errorf("Sorry, something went terribaly wrong")
		}
		agent = parent.Name
	}
	pack, _ := api.Packname(agent).Decode()
	ac, err := r.vars.Assets.FindAgent(r.vars.User.Email, pack)
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

func (r *AIKit) ReadAgentConfig(ctx context.Context, vars *api.Vars, parent *api.Agent, _ *api.ToolFunc, args api.ArgMap) (*api.AppConfig, error) {
	packsub, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	if packsub == "self" {
		if parent == nil {
			return nil, fmt.Errorf("Sorry, something went terribaly wrong")
		}
		packsub = parent.Pack + "/" + parent.Name
	}

	pn := api.Packname(packsub).Clean()
	pack, name := pn.Decode()
	args["kit"] = "agent"
	args["pack"] = pack
	args["name"] = name

	var loader = NewConfigLoader(r.vars)

	// load from script
	// TODO only load mime-type yaml
	if v := args.GetString("script"); v != "" {
		if strings.HasSuffix(v, ".yaml") || strings.HasSuffix(v, ".yml") {
			if err := loader.LoadContent(v); err != nil {
				// return nil, err
			}
		}
	}

	config, err := loader.LoadAgentConfig(pn)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func (r *AIKit) TransferAgent(_ context.Context, _ *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (*api.Result, error) {
	agent, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}

	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}

// agent can be invoked directly by name with kit name "agent:"
// /agent:pack/sub and agent__pack__sub as tool id by LLM
// this tool serves as a simple interface for LLM tool calls.
// agent is required.
// actions defatult to the following if not set:
// entrypoint: "ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm"
func (r *AIKit) SpawnAgent(ctx context.Context, vars *api.Vars, parent *api.Agent, _ *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	packsub, err := api.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	if packsub == "self" {
		if parent == nil {
			return nil, fmt.Errorf("Sorry, something went terribaly wrong")
		}
		packsub = parent.Pack + "/" + parent.Name
	}
	pn := api.Packname(packsub).Clean()
	_, sub := pn.Decode()

	//
	kit := atm.NewSystemKit()
	var entry []string
	if v := args["entrypoint"]; v != nil {
		entry = api.ToStringArray(v)
	}
	if len(entry) == 0 {
		if ac, _ := r.ReadAgentConfig(ctx, vars, parent, nil, args); ac != nil {
			for _, c := range ac.Agents {
				if c.Name == sub {
					entry = c.Entrypoint
					break
				}
			}
		}
	}

	// resolve to avoid infinite loop
	var spawnAgent = []string{"ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm"}
	var resolved []string
	if len(entry) == 0 {
		resolved = spawnAgent
	} else {
		for _, v := range entry {
			if v == "ai:spawn_agent" {
				resolved = append(resolved, spawnAgent...)
			} else {
				resolved = append(resolved, v)
			}
		}
	}
	// FIX: pass on the calling agent.
	result, err := kit.InternalSequence(ctx, vars, "", resolved, args)
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

func (r *AIKit) NewAgent(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	v, err := r.createAgent(ctx, vars, parent, tf, args)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: fmt.Sprintf("agent %s created", v.Name),
	}, nil
}

func (r *AIKit) ReloadAgent(ctx context.Context, _ *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (*api.Result, error) {
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

func (r *AIKit) ListTools(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (string, error) {
	// log.GetLogger(ctx).Debugf("List tools: %s %+v\n", tf, args)

	var user = r.vars.User.Email
	// cached list
	key := ListCacheKey{
		Type: "tool",
		User: user,
	}
	if v, ok := listToolsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached tool list: %+v\n", key)
		return v, nil
	}

	list, count, err := listTools(r.vars.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available tools: %v\n\n%s\n", count, list)
	listToolsCache.Add(key, v)

	return v, nil
}

func (r *AIKit) GetToolInfo(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (string, error) {
	const tpl = `
Tool: %s__%s
Description: %s
Parameters: %s
`
	// log.GetLogger(ctx).Debugf("Tool info: %s:%s %+v\n", tf.Kit, tf.Name, args)

	tid, err := api.GetStrProp("tool", args)
	if err != nil {
		return "", err
	}

	kit, name := api.Kitname(tid).Decode()

	tc, err := r.vars.Assets.FindToolkit(r.vars.User.Email, kit)
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

func (r *AIKit) ReadToolConfig(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args api.ArgMap) (*api.AppConfig, error) {
	tid, err := api.GetStrProp("tool", args)
	if err != nil {
		return nil, err
	}

	kn := api.Kitname(tid).Clean()
	kit, name := kn.Decode()
	args["kit"] = kit
	args["pack"] = ""
	args["name"] = name

	var loader = NewConfigLoader(r.vars)

	if v := args.GetString("script"); v != "" {
		if strings.HasSuffix(v, ".yaml") || strings.HasSuffix(v, ".yml") {
			if err := loader.LoadContent(v); err != nil {
				// return nil, err
			}
		}
	}

	config, err := loader.LoadToolConfig(kn)
	if err != nil {
		return nil, err
	}
	// args["config"] = config
	return config, nil
}

func (r *AIKit) ListModels(ctx context.Context, vars *api.Vars, _ *api.Agent, tf *api.ToolFunc, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List models: %s:%s %+v\n", tf.Kit, tf.Name, args)

	var user = r.vars.User.Email
	// cached list
	key := ListCacheKey{
		Type: "model",
		User: user,
	}
	if v, ok := listToolsCache.Get(key); ok {
		log.GetLogger(ctx).Debugf("Using cached model list: %+v\n", key)
		return v, nil
	}

	list, count, err := listModels(r.vars.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available models: %v\n\n%s\n", count, list)
	listToolsCache.Add(key, v)

	return v, nil
}

func (r *AIKit) ListMessages(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (string, error) {
	// log.GetLogger(ctx).Debugf("List messages: %s:%s %+v\n", tf.Kit, tf.Name, args)

	maxHistory, err := api.GetIntProp("max_history", args)
	if err != nil || maxHistory <= 0 {
		maxHistory = 7
	}
	maxSpan, err := api.GetIntProp("max_span", args)
	if err != nil || maxSpan <= 0 {
		maxSpan = 1440
	}
	// offset, err := api.GetIntProp("offset", args)
	// if err != nil || offset <= 0 {
	// 	offset = 0
	// }
	roles, err := api.GetArrayProp("roles", args)
	if err != nil || len(roles) == 0 {
		roles = []string{"assistant", "user"}
	}

	var option = &api.MemOption{
		MaxHistory: maxHistory,
		MaxSpan:    maxSpan,
		Offset:     0,
		Roles:      roles,
	}
	format, err := api.GetStrProp("format", args)

	history, count, err := loadHistory(r.vars.History, option)

	if err != nil {
		return "", fmt.Errorf("Failed to recall messages (%s): %v", option, err)
	}
	if count == 0 {
		return fmt.Sprintf("No messages (%s)", option), nil
	}
	// if count > 0 {
	// 	log.GetLogger(ctx).Debugf("Recalled %v messages in memory less than %v minutes old\n", count, maxSpan)
	// }

	if format == "json" || format == "application/json" {
		b, err := json.MarshalIndent(history, "", "    ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	var b bytes.Buffer
	for _, v := range history {
		b.WriteString("\n* ROLE: ")
		b.WriteString(v.Role)
		b.WriteString("\n  CONTENT:\n")
		b.WriteString(v.Content)
		b.WriteString("\n\n")
	}
	var v = fmt.Sprintf("Messages (%s): %v\n\n%s\n", option, count, b.String())
	return v, nil
}

func (r *AIKit) SaveMessages(_ context.Context, _ *api.Vars, _ *api.Agent, _ *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	data, err := api.GetStrProp("messages", args)
	args.History()
	if err != nil {
		return nil, err
	}

	var messages []*api.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided for storing.")
	}

	if err := r.vars.History.Save(messages); err != nil {
		return nil, err
	}

	return &api.Result{
		Value: fmt.Sprintf("%v messages saved successfully.", len(messages)),
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
	list, err := r.vars.Assets.ListToolkit(r.vars.User.Email)
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
	for pack, v := range agents {
		for _, sub := range v.Agents {
			key := pack + "/" + sub.Name
			dict[key] = sub
		}
	}

	keys := make([]string, 0)
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(fmt.Sprintf("%s - %s\n\n", k, dict[k].Description))
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
			list = append(list, fmt.Sprintf("%s:%s - %s\n\n", kit, v.Name, v.Description))
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
			list = append(list, fmt.Sprintf("%s/%s - %s\n    %s\n    %s\n    %s\n    %s\n\n", set, level, v.Provider, v.Model, v.BaseUrl, v.ApiKey, v.Description))
		}
	}

	sort.Strings(list)
	return strings.Join(list, "\n"), len(list), nil
}

func loadHistory(store api.MemStore, opt *api.MemOption) ([]*api.Message, int, error) {
	history, err := store.Load(opt)
	if err != nil {
		return nil, 0, err
	}

	count := len(history)
	// if count == 0 {
	// 	return nil, 0, api.NewNotFoundError("no messages")
	// }

	return history, count, nil

	// b, err := json.MarshalIndent(history, "", "    ")
	// if err != nil {
	// 	return "", 0, err
	// }
	// return string(b), count, nil
}

// lookup model from embedded agents first and then from parents
func findModel(a *api.Agent, set, level string) *api.Model {
	if a.Model != nil {
		if set == a.Model.Set && level == a.Model.Level {
			return a.Model
		}
	}
	for _, v := range a.Embed {
		m := findModel(v, set, level)
		if m != nil {
			return m
		}
	}

	// parent
	parent := a.Parent
	for {
		if parent == nil {
			break
		}
		m := findModel(parent, set, level)
		if m != nil {
			return m
		}
		parent = parent.Parent
	}
	return nil
}
