package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

func NewAgentHandler(sw *Swarm) func(*api.Vars, *api.Agent) Handler {
	return func(vars *api.Vars, agent *api.Agent) Handler {
		return &agentHandler{
			vars:  vars,
			agent: agent,
			//
			sw: sw,
		}
	}
}

type agentHandler struct {
	agent *api.Agent
	vars  *api.Vars
	//
	sw *Swarm
}

func (h *agentHandler) Serve(req *api.Request, resp *api.Response) error {
	var r = h.agent
	var ctx = req.Context()

	log.GetLogger(ctx).Debugf("Serve agent: %s\n", r.Name)

	// advices
	noop := func(vars *api.Vars, _ *api.Request, _ *api.Response, _ api.Advice) error {
		return nil
	}
	if r.BeforeAdvice != nil {
		if err := r.BeforeAdvice(h.vars, req, resp, noop); err != nil {
			return err
		}
	}

	if r.AroundAdvice != nil {
		next := func(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
			return h.handle(ctx, req, resp)
		}
		if err := r.AroundAdvice(h.vars, req, resp, next); err != nil {
			return err
		}
	} else {
		if err := h.handle(ctx, req, resp); err != nil {
			return err
		}
	}

	if r.AfterAdvice != nil {
		if err := r.AfterAdvice(h.vars, req, resp, noop); err != nil {
			return err
		}
	}

	return nil
}

func (h *agentHandler) handle(ctx context.Context, req *api.Request, resp *api.Response) error {
	var r = h.agent

	// apply template/load
	apply := func(vars *api.Vars, ext, s string) (string, error) {
		//
		if ext == "tpl" {
			// TODO custom template func?
			return applyTemplate(s, vars, tplFuncMap)
		}
		return s, nil
	}

	var chatID = h.vars.Config.ChatID
	var history []*api.Message

	// 1. New System Message
	// System role prompt as first message
	if r.Instruction != nil {
		// update the request instruction
		content, err := apply(h.vars, r.Instruction.Type, r.Instruction.Content)
		if err != nil {
			return err
		}

		content, err = h.makePrompt(ctx, req, content)
		if err != nil {
			return err
		}

		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    nvl(r.Instruction.Role, api.RoleSystem),
			Content: content,
			Sender:  r.Name,
		})
		log.GetLogger(ctx).Debugf("Added new system role message: %v\n", len(history))
	}

	//
	// prepend message to user query
	var query = req.RawInput.Query()
	if r.Message != "" {
		query = r.Message + "\n" + query
	}

	// 2. Historical Messages - skip system role
	// TODO
	if !r.New && len(h.vars.History) > 0 {
		// msg := h.summarize(ctx, h.vars.History, query)
		log.GetLogger(ctx).Debugf("using %v messaages from history\n", len(h.vars.History))
		for _, msg := range h.vars.History {
			if msg.Role != api.RoleSystem {
				history = append(history, msg)
				log.GetLogger(ctx).Debugf("Added historical non system role message: %v\n", len(history))
			}
		}
	}

	// 3. New User Message
	req.Messages = append(req.Messages, &api.Message{
		ID:      uuid.NewString(),
		ChatID:  chatID,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: query,
		Sender:  r.Name,
	})

	// merge args
	var args map[string]any
	if r.Arguments != nil || req.Arguments != nil {
		args = make(map[string]any)
		maps.Copy(args, r.Arguments)
		maps.Copy(args, req.Arguments)
	}

	history = append(history, req.Messages...)
	log.GetLogger(ctx).Debugf("Added new user role message: %v\n", len(history))

	// Request
	initLen := len(history)

	//
	// var runTool = h.toolCall(h.vars, h.agent)
	var runTool = h.createCaller()

	var model = r.Model
	// resolve if model is @agent
	if strings.HasPrefix(model.Model, "@") {
		if v, err := h.makeModel(ctx, req, model.Model); err != nil {
			return err
		} else {
			model = v
		}
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
		//
		Arguments: args,
		//
		Vars: h.vars,
	}

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
			Role:        nvl(result.Role, api.RoleAssistant),
			Sender:      r.Name,
		}
		// TODO add Value field to message?
		history = append(history, &message)
	}
	//
	h.vars.Extra[extraResult] = result.Result.Value
	h.vars.History = history

	//
	resp.Messages = history[initLen:]
	resp.Agent = r
	resp.Result = result.Result
	return nil
}

// run sub agent with inherited env
func (h *agentHandler) exec(req *api.Request, resp *api.Response) error {
	if err := h.sw.Clone().Run(req, resp); err != nil {
		return err
	}
	if resp.Result == nil {
		return fmt.Errorf("Empty result")
	}
	return nil
}

// dynamically generate prompt if content starts with @<agent>
// otherwise, return s unchanged
func (h *agentHandler) makePrompt(ctx context.Context, parent *api.Request, s string) (string, error) {
	content := strings.TrimSpace(s)
	if !strings.HasPrefix(content, "@") {
		return s, nil
	}
	// @agent instruction...
	agent, instruction := split2(content[1:], " ", "")
	prompt, err := h.callAgent(parent, agent, instruction)

	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Infof("🤖 prompt: %s\n", head(prompt, 100))

	return prompt, nil
}

// dynamcally make LLM model
func (h *agentHandler) makeModel(ctx context.Context, parent *api.Request, s string) (*api.Model, error) {
	agent := strings.TrimPrefix(s, "@")
	out, err := h.callAgent(parent, agent, "")
	if err != nil {
		return nil, err
	}
	var model api.Model
	if err := json.Unmarshal([]byte(out), &model); err != nil {
		return nil, err
	}

	log.GetLogger(ctx).Infof("🤖 model: %s/%s %s\n", model.Provider, model.Model, model.BaseUrl)

	// replace api key
	ak, err := h.sw.Secrets.Get(h.sw.User.Email, model.ApiKey)
	if err != nil {
		return nil, err
	}
	model.ApiKey = ak
	return &model, nil
}

func (h *agentHandler) summarize(ctx context.Context, parent *api.Request, history []*api.Message, agent string) (*api.Message, error) {
	log.GetLogger(ctx).Debugf("using %v messaages from history\n", len(history))
	for _, msg := range history {
		if msg.Role != api.RoleSystem {
			history = append(history, msg)
			log.GetLogger(ctx).Debugf("Added historical non system role message: %v\n", len(history))
		}
	}

	content, err := h.callAgent(parent, agent, "")
	if err != nil {
		return nil, err
	}

	log.GetLogger(ctx).Infof("🤖 context: %s\n", head(content, 100))
	msg := &api.Message{
		Content: content,
		Role:    "",
	}
	return msg, nil
}

func (h *agentHandler) callAgent(parent *api.Request, agent string, prompt string) (string, error) {
	req := parent.Clone()
	req.Agent = agent
	input := req.RawInput.Clone()
	// input.Agent = agent
	// prepend additional instruction to user query
	input.Message = prompt + "\n" + input.Message
	req.RawInput = input

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
