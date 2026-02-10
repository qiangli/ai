package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/util"

	"path/filepath"
)

type AIKit struct {
	vars *api.Vars
}

func NewAIKit(vars *api.Vars) *AIKit {
	return &AIKit{
		vars: vars,
	}
}

type ListCacheKey struct {
	Type string
	User string
}

// var (
// 	listAgentsCache = expirable.NewLRU[ListCacheKey, string](10000, nil, time.Second*900)
// 	listToolsCache  = expirable.NewLRU[ListCacheKey, string](10000, nil, time.Second*900)
// )

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

func (r *AIKit) getAdapter(agent *api.Agent, args map[string]any) (api.LLMAdapter, error) {
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

func (r *AIKit) CallLlm(ctx context.Context, vars *api.Vars, agent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (any, error) {
	var owner = r.vars.User.Email
	var sessionID = r.vars.SessionID

	// check arg for overide
	if v, err := r.checkAndCreate(ctx, vars, agent, tf, args); err == nil {
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
	resolveModel := func(alias string) (*api.Model, error) {
		// set/level
		set, level := api.Setlevel(alias).Decode()
		// embeded/inherited
		if v := findModel(agent, set, level); v != nil {
			return v, nil
		}
		// external
		v, err := conf.LoadModel(owner, set, level, r.vars.Assets)
		if err != nil {
			return nil, err
		}
		return v, nil
	}
	resolveModels := func(aliases []string) ([]*api.Model, error) {
		if len(aliases) == 0 {
			return nil, fmt.Errorf("LLM Model is required")
		}
		var models []*api.Model
		for _, model := range aliases {
			v, err := resolveModel(model)
			if err == nil {
				models = append(models, v)
			}
		}
		if len(models) == 0 {
			return nil, fmt.Errorf("No valid models found for %v", aliases)
		}
		return models, nil
	}

	var models []*api.Model
	if agent.Model != nil {
		models = []*api.Model{agent.Model}
	} else {
		// default
		models = []*api.Model{r.vars.RootAgent.Model}
	}

	// models take precedence over model
	// accepted:
	// a,b,c
	// [a,b,c]
	// json array
	if v, found := args["models"]; found {
		switch vt := v.(type) {
		case string, []any:
			aliases := api.ToStringArray(vt)
			v, err := resolveModels(aliases)
			if err != nil {
				return nil, err
			}
			models = v
		case []string:
			v, err := resolveModels(vt)
			if err != nil {
				return nil, err
			}
			models = v
		default:
			return nil, fmt.Errorf("invalid models format: %+v.", v)
		}
	} else if v, found := args["model"]; found {
		switch vt := v.(type) {
		case *api.Model:
			models = []*api.Model{vt}
		case string, []any:
			aliases := api.ToStringArray(vt)
			v, err := resolveModels(aliases)
			if err != nil {
				return nil, err
			}
			models = v
		case []string:
			v, err := resolveModels(vt)
			if err != nil {
				return nil, err
			}
			models = v
		default:
			return nil, fmt.Errorf("invalid model format: %+v.", v)
		}
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

	// request
	var packname = api.NewPackname(agent.Pack, agent.Name)
	var history []*api.Message

	// 1. New System Message
	// instruction/system role prompt as first message
	if prompt != "" {
		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			Session: sessionID,
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
		Session: sessionID,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: query,
		Sender:  owner,
		Agent:   packname,
	})

	// call LLM
	agent.Prompt = prompt
	agent.History = history
	agent.Query = query
	//
	agent.Tools = tools
	agent.Models = models

	var result *api.Result
	var respErr error
	var sender string

	//
	// Seed the random number generator
	rd := rand.New(rand.NewSource(time.Now().UnixNano()))
	// Shuffle the models slice
	rd.Shuffle(len(models), func(i, j int) {
		models[i], models[j] = models[j], models[i]
	})

	// collect all errors
	var errors []string

	for _, model := range models {
		agent.Model = model

		sender = model.Provider
		//
		result, respErr = r.LlmAdapter(ctx, vars, agent, tf, args)
		if respErr == nil && result != nil {
			break
		}
		errors = append(errors, respErr.Error())
	}

	// report all errors if the last error is not nil/none of the attempts was successful
	if respErr != nil {
		err := strings.Join(errors, ";\n")
		args["error"] = err
		return nil, fmt.Errorf("%s", err)
	}
	if result == nil {
		result = &api.Result{
			Value: "Empty response",
		}
	}
	args["result"] = result.Value

	// NOTE
	// keep query of the current agent but clear the prompt (and history???)
	// This is important for agent transfer and flow:*
	// which may cause the wrong prompt for subsequent agents (with out instrutions)
	delete(args, "prompt")
	// TODO: should be cleared?
	delete(args, "history")

	if result.State == api.StateTransfer {
		if result.NextAgent == "" {
			return nil, fmt.Errorf("agent is required for transfer")
		}
		args["agent"] = result.NextAgent
		return r.SpawnAgent(ctx, vars, agent, tf, args)
	}

	// save assistant response message
	message := api.Message{
		ID:      uuid.NewString(),
		Session: sessionID,
		Created: time.Now(),
		//
		ContentType: result.MimeType,
		Content:     result.Value,
		//
		Role: nvl(result.Role, api.RoleAssistant),
		//
		Sender: sender,
		Agent:  packname,
	}
	history = append(history, &message)

	// save the context
	if err := r.vars.History.Save(history); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *AIKit) LlmAdapter(ctx context.Context, vars *api.Vars, agent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	const prompt = `
	The original request exceeds the maximum input size (%v) and has been rewritten as follows:
	
	## Instruction
	Review the content of the file: %q

	## Context History
	Refer to the background information in the file: %q

	## Query
	Follow the specific request or task outlined in the file: %q

	Please carefully review the contents of these files and provide a detailed response or solution based on the instructions and query provided.
	`
	var owner = r.vars.User.Email

	getToken := func(model *api.Model) (func() string, error) {
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
		return token, nil
	}

	llmAdapter, err := r.getAdapter(agent, args)
	if err != nil {
		return nil, err
	}

	// request
	var req = &api.Request{
		Agent: agent,
	}

	// ensure defaults
	if req.MaxTurns() <= 0 {
		req.SetMaxTurns(api.DefaultMaxTurns)
	}

	// https://platform.openai.com/docs/models/gpt-5-mini
	maxInputSize, _ := api.GetIntProp("max_input_size", args)
	if maxInputSize <= 0 {
		maxInputSize = 272000*4 - 1000
	}
	total := len(agent.Prompt) + len(agent.Query)
	for _, v := range agent.History {
		total += len(v.Content)
	}
	if total < maxInputSize {
		req.Query = agent.Query
		req.Prompt = agent.Prompt
		req.Messages = agent.History
	} else {
		aid := api.NewPackname(agent.Pack, agent.Name).ID()
		rid := uuid.NewString()
		dir := filepath.Join(vars.Roots.Workspace.Path, "oversize", aid, rid)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		insfile := filepath.Join(dir, "instruction.txt")
		ctxfile := filepath.Join(dir, "context.json")
		qfile := filepath.Join(dir, "query.txt")

		os.WriteFile(insfile, []byte(agent.Prompt), 0600)
		var history []byte
		if len(agent.History) == 0 {
			history = []byte("no context")
		} else {
			v, err := json.Marshal(history)
			if err != nil {
				return nil, err
			}
			history = v
		}
		os.WriteFile(insfile, history, 0600)
		os.WriteFile(insfile, []byte(agent.Query), 0600)

		// remove instruction/context and rewrite the query
		req.Prompt = ""
		req.Messages = nil
		req.Query = fmt.Sprintf(prompt, maxInputSize, insfile, ctxfile, qfile)
	}

	//
	req.Arguments = args

	req.Tools = agent.Tools
	req.Runner = agent.Runner

	req.Model = agent.Model

	token, err := getToken(agent.Model)
	if err != nil {
		return nil, err
	}
	req.Token = token

	var resp *api.Response
	resp, err = llmAdapter.Call(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (r *AIKit) ListAgents(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List agents: %s %+v\n", tf, args)

	var user = r.vars.User.Email

	list, count, err := listAgents(r.vars.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available agents: %v\n\n%s\n", count, list)

	return v, nil
}

func (r *AIKit) ListSkills(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("List skills: %s %+v\n", tf, args)
	if vars == nil || vars.Roots.Workspace.Path == "" {
		return "", fmt.Errorf("workspace root not available")
	}

	list, count, err := listSkills(vars.Roots.Workspace.Path)
	if err != nil {
		return "", err
	}

	var v = fmt.Sprintf("Available skills: %v\n\n%s\n", count, list)

	return v, nil
}

func (r *AIKit) GetSkillInfo(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (string, error) {
	log.GetLogger(ctx).Debugf("Get skill info: %s %+v\n", tf, args)
	if vars == nil || vars.Roots.Workspace.Path == "" {
		return "", fmt.Errorf("workspace root not available")
	}

	skill, err := api.GetStrProp("skill", args)
	if err != nil {
		return "", err
	}

	out, err := getSkillInfo(vars.Roots.Workspace.Path, skill)
	if err != nil {
		return "", err
	}
	return out, nil
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
func (r *AIKit) SpawnAgent(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
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

	//
	// resolve spawn_agent to avoid infinite loop
	resolve := func(actions, defaults []string) []string {
		var resolved []string
		if len(actions) == 0 {
			resolved = defaults
		} else {
			for _, v := range actions {
				if v == "ai:spawn_agent" {
					resolved = append(resolved, defaults...)
				} else {
					resolved = append(resolved, v)
				}
			}
		}
		return resolved
	}

	//
	var runner = vars.RootAgent.Runner
	if parent != nil {
		runner = parent.Runner
	}

	sequence := func(actions []string) (*api.Result, error) {
		var result any
		var err error
		for _, v := range actions {
			result, err = runner.Run(ctx, v, args)
		}
		return api.ToResult(result), err
	}

	chain := func(chain, actions []string) (*api.Result, error) {
		final := func() (any, error) {
			if len(actions) == 0 {
				return nil, nil
			}
			return sequence(actions)
		}

		out, err := atm.StartChainActions(ctx, vars, chain, args, final)
		if err != nil {
			return nil, err
		}
		return api.ToResult(out), nil
	}

	//
	agent, err := r.createAgent(ctx, vars, parent, tf, args)
	if err != nil {
		return nil, err
	}
	var entry = agent.Entrypoint
	var before = agent.Before
	var after = agent.After
	var around = agent.Around

	if v := args["entrypoint"]; v != nil {
		entry = api.ToStringArray(v)
	}

	// before advices is intended to prepare and tranform args
	if len(before) > 0 {
		before = resolve(before, nil)
		_, err := sequence(entry)
		if err != nil {
			return nil, err
		}
	}

	var result *api.Result
	// default entrypoint
	// TODO split into before/around/after advices?
	defaultEntry := []string{"ai:new_agent", "ai:build_query", "ai:build_prompt", "ai:build_context", "ai:call_llm"}
	// TODO verify - this optimization is wrong
	// var defaultEntry = []string{"ai:new_agent"}
	// // message is not inherited from embeds
	// if api.IsTemplate(agent.Message) {
	// 	defaultEntry = append(defaultEntry, "ai:build_query")
	// }
	// if len(agent.Embed) > 0 || api.IsTemplate(agent.Instruction) {
	// 	defaultEntry = append(defaultEntry, "ai:build_prompt")
	// }
	// if len(agent.Embed) > 0 || api.IsTemplate(agent.Context) {
	// 	defaultEntry = append(defaultEntry, "ai:build_context")
	// }
	// defaultEntry = append(defaultEntry, "ai:call_llm")

	entry = resolve(entry, defaultEntry)
	around = resolve(around, nil)
	if len(around) > 0 {
		result, err = chain(around, entry)
	} else {
		result, err = sequence(entry)
	}
	if err != nil {
		return nil, err
	}

	// after advices are intended to convert results
	if len(after) > 0 {
		after = resolve(after, nil)
		args["result"] = result
		_, err := sequence(after)
		if err != nil {
			return nil, err
		}
		result = api.ToResult(args["result"])
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

func (r *AIKit) NewAgent(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args api.ArgMap) (any, error) {
	v, err := r.createAgent(ctx, vars, parent, tf, args)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		Value: fmt.Sprintf("agent %s/%s created ✨", v.Pack, v.Name),
	}, nil
}

func (r *AIKit) ListTools(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (string, error) {
	var user = r.vars.User.Email

	list, count, err := listTools(r.vars.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available tools: %v\n\n%s\n", count, list)

	return v, nil
}

func (r *AIKit) GetToolInfo(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (string, error) {
	const tpl = `
Tool: %s__%s
Description: %s
Parameters: %s
`
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

	list, count, err := listModels(r.vars.Assets, user)
	if err != nil {
		return "", err
	}
	var v = fmt.Sprintf("Available models: %v\n\n%s\n", count, list)

	return v, nil
}

func (r *AIKit) ListMessages(ctx context.Context, vars *api.Vars, _ *api.Agent, _ *api.ToolFunc, args map[string]any) (string, error) {
	maxHistory, err := api.GetIntProp("max_history", args)
	if err != nil || maxHistory <= 0 {
		maxHistory = api.DefaultMaxHistory
	}
	maxSpan, err := api.GetIntProp("max_span", args)
	if err != nil || maxSpan <= 0 {
		maxSpan = api.DefaultMaxSpan
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
	format, err := api.GetStrProp("format", args)

	history, count, err := loadHistory(r.vars.History, option)

	if err != nil {
		return "", fmt.Errorf("Failed to recall messages (%s): %v", option, err)
	}
	if count == 0 {
		return fmt.Sprintf("No messages (%s)", option), nil
	}

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
		b.WriteString("\n  CREATED: ")
		b.WriteString(fmt.Sprintf("%v", v.Created))
		b.WriteString("\n  AGENT: ")
		b.WriteString(fmt.Sprintf("%v", v.Agent))
		b.WriteString("\n\n")
	}
	var v = fmt.Sprintf("Messages (%s): %v\n\n%s\n", option, count, b.String())
	return v, nil
}

func (r *AIKit) SaveMessages(_ context.Context, _ *api.Vars, _ *api.Agent, _ *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	data, err := api.GetStrProp("messages", args)
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

func (r *AIKit) Sleep(_ context.Context, _ *api.Vars, _ *api.Agent, _ *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	duration, err := api.GetStrProp("duration", args)
	if err != nil {
		return nil, err
	}
	sec, err := util.ParseDuration(duration)
	if err != nil {
		return nil, err
	}
	// optional
	msg, _ := api.GetStrProp("report", args)

	time.Sleep(sec)

	return &api.Result{
		Value: msg,
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
				Output: v.Output,
				//
				Arguments: v.Arguments,
				// Provider: nvl(v.Provider, tc.Provider),
				// BaseUrl:  nvl(v.BaseUrl, tc.BaseUrl),
				// ApiKey:   nvl(v.ApiKey, tc.ApiKey),
				//
			}
			tools = append(tools, tool)
		}
	}
	return tools, nil
}

type SkillEntry struct {
	Name        string
	Description string
	Path        string
}

type SkillInfo struct {
	Name        string
	Description string
	Instruction string
	Tools       []string
	References  []string
	Scripts     []string
	Assets      []string
	Path        string
}

func getSkillInfo(workspace string, skillName string) (string, error) {
	if strings.TrimSpace(skillName) == "" {
		return "", fmt.Errorf("skill is required")
	}

	// Load roots from <workspace>/skills/config.md (same as listSkills)
	cfgPath := filepath.Join(workspace, "skills", "config.md")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(b), "\n")
	var roots []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		if strings.HasPrefix(ln, "- ") {
			p := strings.TrimSpace(strings.TrimPrefix(ln, "- "))
			if p != "" {
				roots = append(roots, p)
			}
		}
	}

	want := strings.ToLower(strings.TrimSpace(skillName))
	var matchedDir string
	var matchedMeta struct {
		Name        string
		Description string
		Instruction string
	}

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		items, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, it := range items {
			if !it.IsDir() {
				continue
			}
			base := strings.TrimSpace(it.Name())
			if base == "" || strings.HasPrefix(base, ".") {
				continue
			}
			skillDir := filepath.Join(root, base)
			skillMd := filepath.Join(skillDir, "SKILL.md")
			mb, err := os.ReadFile(skillMd)
			if err != nil {
				continue
			}

			name, desc, instr, ok := parseSkillMd(string(mb))
			if !ok {
				continue
			}
			if strings.ToLower(strings.TrimSpace(name)) == want {
				matchedDir = skillDir
				matchedMeta.Name = name
				matchedMeta.Description = desc
				matchedMeta.Instruction = instr
				break
			}
		}
		if matchedDir != "" {
			break
		}
	}

	if matchedDir == "" {
		return "", fmt.Errorf("skill not found: %s", skillName)
	}

	tools, refs, scripts, assets := listSkillFolderSections(matchedDir)

	info := SkillInfo{
		Name:        matchedMeta.Name,
		Description: matchedMeta.Description,
		Instruction: matchedMeta.Instruction,
		Tools:       tools,
		References:  refs,
		Scripts:     scripts,
		Assets:      assets,
		Path:        matchedDir,
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Skill: %s\n", info.Name))
	buf.WriteString(fmt.Sprintf("Description: %s\n", info.Description))
	buf.WriteString("Instruction:\n")
	if strings.TrimSpace(info.Instruction) == "" {
		buf.WriteString("(none)\n")
	} else {
		buf.WriteString(info.Instruction)
		if !strings.HasSuffix(info.Instruction, "\n") {
			buf.WriteString("\n")
		}
	}
	buf.WriteString("\nTools:\n")
	buf.WriteString(formatList(info.Tools))
	buf.WriteString("\nReferences:\n")
	buf.WriteString(formatList(info.References))
	buf.WriteString("\nScripts:\n")
	buf.WriteString(formatList(info.Scripts))
	buf.WriteString("\nAssets:\n")
	buf.WriteString(formatList(info.Assets))
	buf.WriteString(fmt.Sprintf("\nPath: %s\n", info.Path))

	return buf.String(), nil
}

func parseSkillMd(content string) (name string, desc string, instr string, ok bool) {
	content = strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(content, "---") {
		return "", "", "", false
	}
	rest := strings.TrimPrefix(content, "---")
	if strings.HasPrefix(rest, "\r\n") {
		rest = strings.TrimPrefix(rest, "\r\n")
	} else if strings.HasPrefix(rest, "\n") {
		rest = strings.TrimPrefix(rest, "\n")
	}
	end := strings.Index(rest, "\n---")
	if end < 0 {
		end = strings.Index(rest, "\r\n---")
	}
	if end < 0 {
		return "", "", "", false
	}
	fm := rest[:end]
	body := rest[end:]
	// drop closing --- line
	if i := strings.Index(body, "---"); i >= 0 {
		body = body[i+3:]
	}
	body = strings.TrimLeft(body, "\r\n")
	instr = strings.TrimSpace(body)

	for _, fl := range strings.Split(fm, "\n") {
		fl = strings.TrimSpace(fl)
		if fl == "" || strings.HasPrefix(fl, "#") {
			continue
		}
		k, v, has := strings.Cut(fl, ":")
		if !has {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.Trim(strings.TrimSpace(v), "\"'")
		switch k {
		case "name":
			name = v
		case "description":
			desc = v
		}
	}
	if name == "" {
		return "", "", "", false
	}
	return name, desc, instr, true
}

func listSkillFolderSections(skillDir string) (tools []string, refs []string, scripts []string, assets []string) {
	// Heuristic mapping to common AgentSkills structure.
	// We'll list immediate children of these folders if they exist.
	folders := []struct {
		name string
		out  *[]string
	}{
		{"tools", &tools},
		{"references", &refs},
		{"scripts", &scripts},
		{"assets", &assets},
	}
	for _, f := range folders {
		p := filepath.Join(skillDir, f.name)
		st, err := os.Stat(p)
		if err != nil || !st.IsDir() {
			continue
		}
		items, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, it := range items {
			nm := it.Name()
			if nm == "" || strings.HasPrefix(nm, ".") {
				continue
			}
			// Add a trailing slash to signal directories.
			if it.IsDir() {
				nm = nm + "/"
			}
			(*f.out) = append((*f.out), nm)
		}
		sort.Strings(*f.out)
	}
	return tools, refs, scripts, assets
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)\n"
	}
	var buf strings.Builder
	for _, it := range items {
		buf.WriteString("- ")
		buf.WriteString(it)
		buf.WriteString("\n")
	}
	return buf.String()
}

func listSkills(workspace string) (string, int, error) {
	// Read skill roots from: <workspace>/skills/config.md
	cfgPath := filepath.Join(workspace, "skills", "config.md")
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", 0, err
	}
	lines := strings.Split(string(b), "\n")
	var roots []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		if strings.HasPrefix(ln, "- ") {
			p := strings.TrimSpace(strings.TrimPrefix(ln, "- "))
			if p != "" {
				roots = append(roots, p)
			}
		}
	}

	entries := make([]SkillEntry, 0)
	seen := make(map[string]struct{})

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		// list immediate subdirectories under root
		items, err := os.ReadDir(root)
		if err != nil {
			// skip non-existent/inaccessible roots
			continue
		}
		for _, it := range items {
			if !it.IsDir() {
				continue
			}
			base := strings.TrimSpace(it.Name())
			if base == "" || strings.HasPrefix(base, ".") {
				continue
			}
			skillDir := filepath.Join(root, base)
			if _, ok := seen[skillDir]; ok {
				continue
			}
			seen[skillDir] = struct{}{}

			skillMd := filepath.Join(skillDir, "SKILL.md")
			mb, err := os.ReadFile(skillMd)
			if err != nil {
				continue
			}

			// Parse YAML front matter only (between leading --- and next ---)
			content := string(mb)
			content = strings.TrimPrefix(content, "\ufeff")
			if !strings.HasPrefix(content, "---") {
				continue
			}
			rest := strings.TrimPrefix(content, "---")
			// allow optional CRLF
			if strings.HasPrefix(rest, "\r\n") {
				rest = strings.TrimPrefix(rest, "\r\n")
			} else if strings.HasPrefix(rest, "\n") {
				rest = strings.TrimPrefix(rest, "\n")
			}
			end := strings.Index(rest, "\n---")
			if end < 0 {
				end = strings.Index(rest, "\r\n---")
			}
			if end < 0 {
				continue
			}
			fm := rest[:end]
			var meta struct {
				Name        string `json:"name" yaml:"name"`
				Description string `json:"description" yaml:"description"`
			}
			// Use a tiny YAML subset parser: key: value lines.
			// (Avoid pulling in a full YAML dep here.)
			for _, fl := range strings.Split(fm, "\n") {
				fl = strings.TrimSpace(fl)
				if fl == "" || strings.HasPrefix(fl, "#") {
					continue
				}
				k, v, ok := strings.Cut(fl, ":")
				if !ok {
					continue
				}
				k = strings.TrimSpace(k)
				v = strings.TrimSpace(v)
				v = strings.Trim(v, "\"'")
				switch strings.ToLower(k) {
				case "name":
					meta.Name = v
				case "description":
					meta.Description = v
				}
			}
			if meta.Name == "" || meta.Description == "" {
				continue
			}

			entries = append(entries, SkillEntry{
				Name:        meta.Name,
				Description: meta.Description,
				Path:        skillDir,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		a := strings.ToLower(entries[i].Name)
		b := strings.ToLower(entries[j].Name)
		if a == b {
			return entries[i].Name < entries[j].Name
		}
		return a < b
	})

	var buf strings.Builder
	for _, e := range entries {
		buf.WriteString(fmt.Sprintf("%s - %s\n[%s]\n\n", e.Name, e.Description, e.Path))
	}
	return buf.String(), len(entries), nil
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

	return history, count, nil
}

// lookup model from embedded agents first and then from parents
func findModel(a *api.Agent, set, level string) *api.Model {
	if a.Model != nil {
		if set == a.Model.Set && level == a.Model.Level {
			return a.Model
		}
	}
	// lookup in ac.Models
	if a.Config != nil {
		ac := a.Config
		if set == ac.Set {
			for k, v := range ac.Models {
				if k == level {
					m := &api.Model{
						Set:   set,
						Level: level,
						//
						Model: v.Model,
						//
						Provider: nvl(v.Provider, ac.Provider),
						BaseUrl:  nvl(v.BaseUrl, ac.BaseUrl),
						ApiKey:   nvl(v.ApiKey, ac.ApiKey),
					}
					return m
				}
			}
		}
	}
	//
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
