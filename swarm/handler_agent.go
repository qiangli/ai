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
	// noop := func(vars *api.Vars, _ *api.Request, _ *api.Response, _ api.Advice) error {
	// 	return nil
	// }
	// if r.BeforeAdvice != nil {
	// 	if err := r.BeforeAdvice(h.vars, req, resp, noop); err != nil {
	// 		return err
	// 	}
	// }

	// if r.AroundAdvice != nil {
	// 	next := func(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
	// 		return h.handle(ctx, req, resp)
	// 	}
	// 	if err := r.AroundAdvice(h.vars, req, resp, next); err != nil {
	// 		return err
	// 	}
	// } else {
	// 	if err := h.handle(ctx, req, resp); err != nil {
	// 		return err
	// 	}
	// }

	// if r.AfterAdvice != nil {
	// 	if err := r.AfterAdvice(h.vars, req, resp, noop); err != nil {
	// 		return err
	// 	}
	// }

	if err := h.handle(ctx, req, resp); err != nil {
		return err
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

	var chatID = h.vars.ChatID
	var history []*api.Message

	// 1. New System Message
	// system role prompt as first message
	if r.Instruction != nil {
		// update the request instruction
		content, err := apply(h.vars, r.Instruction.Type, r.Instruction.Content)
		if err != nil {
			return err
		}

		// dynamic @prompt if requested
		content, err = h.makePrompt(ctx, req, content)
		if err != nil {
			return err
		}

		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: content,
			Sender:  r.Name,
		})
		log.GetLogger(ctx).Debugf("Added system role message: %v\n", len(history))
	}
	// Additional system message
	// developer (openai)| model (gemini)| system (anthropic)
	if r.Message != "" {
		req.Messages = append(req.Messages, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: r.Message,
			Sender:  r.Name,
		})
	}

	// 2. Historical Messages
	// support dynamic context history
	// skip system role
	if !r.New {
		var list []*api.Message
		var emoji = "•"
		if r.Context != "" {
			list, _ = h.contextHistory(ctx, req, r.Context, req.RawInput.Query())
			emoji = "🤖"
		} else {
			list = h.vars.History
		}
		if len(list) > 0 {
			log.GetLogger(ctx).Infof("%s context messages: %v\n", emoji, len(list))
			for i, msg := range list {
				if msg.Role != api.RoleSystem {
					history = append(history, msg)
					log.GetLogger(ctx).Debugf("Added historical message: %v %s %s\n", i, msg.Role, head(msg.Content, 100))
				}
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
		Content: req.RawInput.Query(),
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
	log.GetLogger(ctx).Debugf("Added user role message: %v\n", len(history))

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
		// agent tool
		Arguments: args,
		//
		Vars: h.vars,
	}

	// openai/tts
	if r.Instruction != nil {
		request.Instruction = r.Instruction.Content
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
	// prevent loop
	// TODO support recursion?
	if h.agent.Name == req.Agent {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself.", req.Agent))
	}

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
	agent, instruction := split2(content, " ", "")
	prompt, err := h.callAgent(parent, agent, instruction)

	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Infof("🤖 prompt: %s\n", head(prompt, 100))

	return prompt, nil
}

// dynamcally make LLM model
func (h *agentHandler) makeModel(ctx context.Context, parent *api.Request, agent string) (*api.Model, error) {
	out, err := h.callAgent(parent, agent, "")
	if err != nil {
		return nil, err
	}
	var model api.Model
	if err := json.Unmarshal([]byte(out), &model); err != nil {
		return nil, err
	}

	log.GetLogger(ctx).Infof("🤖 model: %s/%s\n", model.Provider, model.Model)

	// replace api key
	ak, err := h.sw.Secrets.Get(h.sw.User.Email, model.ApiKey)
	if err != nil {
		return nil, err
	}
	model.ApiKey = ak
	return &model, nil
}

func (h *agentHandler) contextHistory(ctx context.Context, parent *api.Request, agent, query string) ([]*api.Message, error) {
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

func (h *agentHandler) callAgent(parent *api.Request, s string, prompt string) (string, error) {
	agent := strings.TrimPrefix(s, "@")

	req := parent.Clone()
	req.Parent = h.agent
	req.Agent = agent
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
