package swarm

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

func NewAgentHandler(auth *api.User, secrets api.SecretStore, tools api.ToolSystem, adapters llm.AdapterRegistry) func(*api.Vars, *api.Agent) Handler {
	return func(vars *api.Vars, agent *api.Agent) Handler {
		var toolCall = atm.NewToolCaller(auth, agent.Owner, secrets, tools)
		return &agentHandler{
			vars:  vars,
			agent: agent,
			//
			user:     auth,
			tools:    tools,
			adapters: adapters,
			toolCall: toolCall,
		}
	}
}

type agentHandler struct {
	agent *api.Agent
	vars  *api.Vars

	//
	user     *api.User
	tools    api.ToolSystem
	adapters llm.AdapterRegistry

	toolCall api.ToolCaller
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
			return h.runLoop(ctx, req, resp)
		}
		if err := r.AroundAdvice(h.vars, req, resp, next); err != nil {
			return err
		}
	} else {
		if err := h.runLoop(ctx, req, resp); err != nil {
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

func (h *agentHandler) runLoop(ctx context.Context, req *api.Request, resp *api.Response) error {
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

		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    nvl(r.Instruction.Role, api.RoleSystem),
			Content: content,
			Sender:  r.Name,
			// Models:  h.vars.Config.Models,
		})
		log.GetLogger(ctx).Debugf("Added new system role message: %v\n", len(history))
	}

	// 2. Historical Messages - skip system role
	// TODO
	if !r.New && len(h.vars.History) > 0 {
		log.GetLogger(ctx).Debugf("using %v messaages from history\n", len(h.vars.History))
		for _, msg := range h.vars.History {
			if msg.Role != api.RoleSystem {
				history = append(history, msg)
				log.GetLogger(ctx).Debugf("Added historical non system role message: %v\n", len(history))
			}
		}
	}

	// 3. New User Message
	if req.Messages == nil {
		// contentType/content
		// messages, err := req.RawInput.FileMessages()
		// if err != nil {
		// 	return err
		// }
		// for i, v := range messages {
		// 	v.ID = uuid.NewString()
		// 	v.ChatID = chatID
		// 	v.Created = time.Now()
		// 	//
		// 	v.Role = api.RoleUser
		// 	v.Sender = r.Name
		// 	v.Models = h.vars.Config.Models
		// 	messages[i] = v
		// }

		// req.Messages = messages
	}
	// prepend message to user query
	var query = req.RawInput.Query()
	if r.Message != "" {
		query = r.Message + "\n" + query
	}
	req.Messages = append(req.Messages, &api.Message{
		ID:      uuid.NewString(),
		ChatID:  chatID,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: query,
		Sender:  r.Name,
	})

	history = append(history, req.Messages...)
	log.GetLogger(ctx).Debugf("Added new user role message: %v\n", len(history))

	// Request
	initLen := len(history)

	//
	runTool := h.toolCall(h.vars, h.agent)

	var request = llm.Request{
		Agent:    r.Name,
		Model:    r.Model,
		Messages: history,
		MaxTurns: r.MaxTurns,
		RunTool:  runTool,
		Tools:    r.Tools,
		//
		Vars: h.vars,
	}

	var adapter llm.LLMAdapter = adapter.Chat
	if h.agent.Adapter != "" {
		if v, err := h.adapters.Get(h.agent.Adapter); err == nil {
			adapter = v
		} else {
			return err
		}
	}

	result, err := adapter(ctx, &request)
	if err != nil {
		return err
	}

	// Response
	//
	if result.Result == nil || result.Result.State != api.StateTransfer {
		message := api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			ContentType: result.ContentType,
			Role:        nvl(result.Role, api.RoleAssistant),
			Content:     result.Content,
			Sender:      r.Name,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	//
	h.vars.Extra[extraResult] = result.Content

	// TODO merge Agent type with api.User
	resp.Agent = &api.Agent{
		Name:    r.Name,
		Display: r.Display,
	}
	resp.Result = result.Result

	//
	h.vars.History = history
	return nil
}
