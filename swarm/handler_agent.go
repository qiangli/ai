package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

func NewAgentHandler(sw *Swarm) (func(*api.Agent) Handler, error) {
	var fm = sprig.FuncMap()
	maps.Copy(fm, tplFuncMap)
	fm["user"] = func() *api.User {
		return sw.User
	}

	tpl := template.New("swarm").Funcs(fm)

	return func(agent *api.Agent) Handler {
		return &agentHandler{
			agent:    agent,
			sw:       sw,
			template: tpl,
		}
	}, nil
}

type agentHandler struct {
	agent    *api.Agent
	sw       *Swarm
	template *template.Template
}

func (h *agentHandler) Serve(req *api.Request, resp *api.Response) error {
	var ctx = req.Context()
	log.GetLogger(ctx).Debugf("Serve agent: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)

	// this needs to happen before everything else
	h.sw.Vars.Global.Set(globalQuery, req.RawInput.Query())
	if err := h.setGlobalEnv(req); err != nil {
		return err
	}

	if err := h.doAgentFlow(req, resp); err != nil {
		h.sw.Vars.Global.Set(globalResult, err.Error())
		return err
	}

	// if result is json, unpack for subsequnent agents/actions
	if len(resp.Result.Value) > 0 {
		var resultMap = make(map[string]any)
		if err := json.Unmarshal([]byte(resp.Result.Value), &resultMap); err == nil {
			h.sw.Vars.Global.Add(resultMap)
		}
	}
	h.sw.Vars.Global.Set(globalResult, resp.Result.Value)

	log.GetLogger(ctx).Debugf("completed: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)
	return nil
}

// run agent first if there is instruction followed by the flow.
// otherwise, run the flow only
func (h *agentHandler) doAgentFlow(req *api.Request, resp *api.Response) error {
	// run agent
	if h.agent.Instruction != nil && h.agent.Instruction.Content != "" {
		if err := h.doAgent(req, resp); err != nil {
			return err
		}
	}

	// flow control agent
	if h.agent.Flow != nil {
		if len(h.agent.Flow.Actions) == 0 && len(h.agent.Flow.Script) == 0 {
			return fmt.Errorf("missing actions or script in flow")
		}
		switch h.agent.Flow.Type {
		case api.FlowTypeSequence:
			if err := h.flowSequence(req, resp); err != nil {
				return err
			}
		case api.FlowTypeParallel:
			if err := h.flowParallel(req, resp); err != nil {
				return err
			}
		case api.FlowTypeChoice:
			if err := h.flowChoice(req, resp); err != nil {
				return err
			}
		case api.FlowTypeMap:
			if err := h.flowMap(req, resp); err != nil {
				return err
			}
		case api.FlowTypeLoop:
			if err := h.flowLoop(req, resp); err != nil {
				return err
			}
		case api.FlowTypeReduce:
			if err := h.flowReduce(req, resp); err != nil {
				return err
			}
		case api.FlowTypeShell:
			if err := h.flowShell(req, resp); err != nil {
				return err
			}
		default:
			return fmt.Errorf("not supported yet %v", h.agent.Flow)
		}
	}

	return nil
}

// copy values from src to dst after calling @agent and applying template if required
func (h *agentHandler) mapAssign(req *api.Request, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if !override {
			if _, ok := dst[key]; ok {
				continue
			}
		}
		// @agent value support
		if v, ok := val.(string); ok {
			if resolved, err := h.resolveArgument(req, v); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		// go template value support
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			if resolved, err := h.applyTemplate(v, dst); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		dst[key] = val
	}
	return nil
}

// create a copy of current global vars
// merge agent environment, update with values from agent arguments if non existant
// support @agent call and go template as value
func (h *agentHandler) setGlobalEnv(req *api.Request) error {
	// var ctx = req.Context()
	// merge request args
	var env = make(map[string]any)
	// copy globals including agent args
	h.sw.Vars.Global.Copy(env)

	// agent global env takes precedence
	if h.agent.Environment != nil {
		h.mapAssign(req, env, h.agent.Environment, true)
	}

	// set agent and req defaults
	// set only when the key does not exist
	if h.agent.Arguments != nil {
		h.mapAssign(req, env, h.agent.Arguments, false)
	}

	// if req.Arguments != nil {
	// 	copy(env, req.Arguments, false)
	// }

	// add query
	// if req.RawInput != nil {
	// 	env["query"] = req.RawInput.Query()
	// }

	h.sw.Vars.Global.Add(env)

	log.GetLogger(req.Context()).Debugf("global env: %+v\n", env)
	return nil
}

func (h *agentHandler) globalEnv() map[string]any {
	var env = make(map[string]any)
	h.sw.Vars.Global.Copy(env)
	return env
}

func (h *agentHandler) doAgent(req *api.Request, resp *api.Response) error {
	var ctx = req.Context()
	var r = h.agent

	env := h.globalEnv()
	h.mapAssign(req, env, req.Arguments, false)

	// apply template/load
	// TODO  vars -> data may break some existing config
	applyGlobal := func(ext, s string) (string, error) {
		if strings.HasPrefix(s, "#!") {
			parts := strings.SplitN(s, "\n", 2)
			if len(parts) == 2 {
				// remove hashbang line
				// return h.applyGlobal(parts[1])
				return h.applyTemplate(parts[1], env)
			}
			// remove hashbang
			return h.applyTemplate(parts[0][2:], env)
		}
		if ext == "tpl" {
			return h.applyTemplate(s, env)
		}
		return s, nil
	}

	resolveGlobal := func(ext, s string) (string, error) {
		// update the request instruction
		content, err := applyGlobal(ext, s)
		if err != nil {
			return "", err
		}

		// dynamic @prompt if requested
		content, err = h.resolvePrompt(ctx, req, content)
		if err != nil {
			return "", err
		}
		return content, nil
	}

	var chatID = h.sw.ChatID
	var history []*api.Message
	var instructions []string

	// 1. New System Message
	// system role prompt as first message
	// inherit embedded agent instructions
	addContext := func(in *api.Instruction, sender string) error {
		content, err := resolveGlobal(in.Type, in.Content)
		if err != nil {
			return err
		}

		// update instruction
		// r.Instruction.Content = content
		instructions = append(instructions, content)

		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: content,
			Sender:  sender,
		})
		log.GetLogger(ctx).Debugf("Added system role message: %v\n", len(history))
		return nil
	}
	for _, v := range r.Embed {
		if v.Instruction != nil {
			addContext(v.Instruction, r.Name)
		}
	}
	if r.Instruction != nil {
		addContext(r.Instruction, r.Name)
	}

	// 2. Historical Messages
	// support dynamic context history
	// skip system role
	if !r.New {
		var list []*api.Message
		var emoji = "•"
		if r.Context != "" {
			// continue without context if failed
			if resolved, err := h.mustResolveContext(ctx, req, r.Context); err != nil {
				log.GetLogger(ctx).Errorf("failed to resolve context %s: %v\n", r.Context, err)
			} else {
				list = resolved
				emoji = "🤖"
			}
		} else {
			list = h.sw.Vars.History
		}
		if len(list) > 0 {
			log.GetLogger(ctx).Infof("%s context messages: %v\n", emoji, len(list))
			for i, msg := range list {
				if msg.Role != api.RoleSystem {
					history = append(history, msg)
					log.GetLogger(ctx).Debugf("Added historical message [%v]: %s %s\n", i, msg.Role, head(msg.Content, 100))
				}
			}
		}
	}

	// 3. New User Message
	// Additional user message
	// embeded messages not inherited for now
	if r.Message != "" {
		msg, err := resolveGlobal("", r.Message)
		if err != nil {
			return err
		}
		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    api.RoleUser,
			Content: msg,
			Sender:  r.Name,
		})
	}

	history = append(history, &api.Message{
		ID:      uuid.NewString(),
		ChatID:  chatID,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: req.RawInput.Query(),
		Sender:  r.Name,
	})

	log.GetLogger(ctx).Debugf("Added messages: %v\n", len(history))

	// Request
	initLen := len(history)

	//
	var runTool = h.createCaller(h.sw.User)

	// resolve if model is @agent
	var model *api.Model
	if v, err := h.resolveModel(ctx, req, r.Model); err != nil {
		return err
	} else {
		model = v
	}

	ak, err := h.sw.Secrets.Get(h.agent.Owner, model.ApiKey)
	if err != nil {
		return err
	}
	token := func() string {
		return ak
	}

	var request = llm.Request{
		Agent:    r.Name,
		Messages: history,
		MaxTurns: r.MaxTurns,
		Tools:    r.Tools,
		//
		Model: model,
		//
		RunTool: runTool,
		// agent tool
		Arguments: env,
		//
		Vars: h.sw.Vars,
		//
		Token: token,
	}

	// openai/tts
	if len(instructions) > 0 {
		request.Instruction = strings.Join(instructions, "\n")
	}
	request.Query = r.RawInput.Query()

	var adapter llm.LLMAdapter = adapter.Chat
	if h.agent.Adapter != "" {
		if v, err := h.sw.Adapters.Get(h.agent.Adapter); err == nil {
			adapter = v
		} else {
			return err
		}
	}

	// LLM adapter
	// TODO model <-> adapter
	result, err := adapter(ctx, &request)
	if err != nil {
		return err
	}
	if result.Result == nil {
		return fmt.Errorf("Empty response")
	}

	// Response
	if result.Result.State != api.StateTransfer {
		message := api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			ContentType: result.Result.MimeType,
			Content:     result.Result.Value,
			// TODO encode result.Result.Content
			Role:   nvl(result.Role, api.RoleAssistant),
			Sender: r.Name,
		}
		// TODO add Value field to message?
		history = append(history, &message)
	}

	h.sw.Vars.History = history
	//
	resp.Messages = history[initLen:]
	resp.Agent = r
	resp.Result = result.Result
	return nil
}

// run sub agent with inherited env
func (h *agentHandler) exec(req *api.Request, resp *api.Response) error {
	// prevent loop
	// TODO support recursion?
	if h.agent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself.", req.Name))
	}

	if err := h.sw.Run(req, resp); err != nil {
		return err
	}
	if resp.Result == nil {
		return fmt.Errorf("Empty result")
	}
	return nil
}

// dynamically generate prompt if content starts with @<agent>
// otherwise, return s unchanged
func (h *agentHandler) resolvePrompt(ctx context.Context, parent *api.Request, s string) (string, error) {
	agent, query, found := parseAgentCommand(s)
	if !found {
		return s, nil
	}
	prompt, err := h.callAgent(parent, agent, query)
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Infof("🤖 prompt: %s\n", head(prompt, 100))

	return prompt, nil
}

// dynamcally make LLM model; return s as is if not an agent command
func (h *agentHandler) resolveModel(ctx context.Context, parent *api.Request, m *api.Model) (*api.Model, error) {
	if m == nil {
		return nil, fmt.Errorf("missling model")
	}
	agent, query, found := parseAgentCommand(m.Model)
	if !found {
		return m, nil
	}
	out, err := h.callAgent(parent, agent, query)
	if err != nil {
		return nil, err
	}
	var model api.Model
	if err := json.Unmarshal([]byte(out), &model); err != nil {
		return nil, err
	}

	log.GetLogger(ctx).Infof("🤖 model: %s/%s\n", model.Provider, model.Model)

	// // replace api key
	// ak, err := h.sw.Secrets.Get(h.sw.User.Email, model.ApiKey)
	// if err != nil {
	// 	return nil, err
	// }
	// model.ApiKey = ak
	return &model, nil
}

func (h *agentHandler) mustResolveContext(ctx context.Context, parent *api.Request, s string) ([]*api.Message, error) {
	agent, query, found := parseAgentCommand(s)
	if !found {
		return nil, fmt.Errorf("invalid context: %s", s)
	}
	out, err := h.callAgent(parent, agent, query)
	if err != nil {
		return nil, err
	}

	var list []*api.Message
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, err
	}

	log.GetLogger(ctx).Debugf("dynamic context messages: (%v) %s\n", len(list), head(out, 100))
	return list, nil
}

// call agent if found. otherwise return s as is
func (h *agentHandler) resolveArgument(parent *api.Request, s string) (any, error) {
	agent, query, found := parseAgentCommand(s)
	if !found {
		return s, nil
	}
	out, err := h.callAgent(parent, agent, query)
	if err != nil {
		return nil, err
	}

	type ArgResult struct {
		Result string
		Error  string
	}

	var arg ArgResult
	if err := json.Unmarshal([]byte(out), &arg); err != nil {
		return nil, err
	}
	if arg.Error != "" {
		return nil, fmt.Errorf("failed resolve argument: %s", arg.Error)
	}
	return arg.Result, nil
}

func (h *agentHandler) callAgent(parent *api.Request, s string, prompt string) (string, error) {
	name := strings.TrimPrefix(s, "@")

	req := parent.Clone()
	req.Parent = h.agent
	req.Name = name
	// prepend additional instruction to user query
	if len(prompt) > 0 {
		req.RawInput.Message = prompt + "\n" + req.RawInput.Message
	}

	resp := &api.Response{}

	err := h.exec(req, resp)
	if err != nil {
		return "", err
	}
	if resp.Result == nil {
		return "", fmt.Errorf("empty response")
	}
	return resp.Result.Value, nil
}

func (h *agentHandler) applyTemplate(tpl string, data any) (string, error) {
	t, err := h.template.Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// func (h *agentHandler) applyGlobal(tpl string) (string, error) {
// 	var out string
// 	fn := func(data map[string]any) error {
// 		if v, err := h.applyTemplate(tpl, data); err != nil {
// 			return err
// 		} else {
// 			out = v
// 		}
// 		return nil
// 	}
// 	if err := h.sw.Vars.Global.Apply(fn); err != nil {
// 		return "", err
// 	}
// 	return out, nil
// }
