package swarm

import (
	"context"
	"fmt"
	"maps"
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
			//
			toolCall: NewToolCaller(sw),
		}
	}
}

type agentHandler struct {
	agent *api.Agent
	vars  *api.Vars

	//
	sw *Swarm

	//
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
	//
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
	var runTool = h.toolCall(h.vars, h.agent)

	var request = llm.Request{
		Agent:    r.Name,
		Model:    r.Model,
		Messages: history,
		MaxTurns: r.MaxTurns,
		Tools:    r.Tools,
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
	result, err := adapter(ctx, &request)
	if err != nil {
		return err
	}
	if result.Result == nil {
		return fmt.Errorf("empty response")
	}

	// Response
	if result.Result.State != api.StateTransfer {
		message := api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			ContentType: result.Result.MimeType,
			Role:        nvl(result.Role, api.RoleAssistant),
			Content:     result.Result.Value,
			Sender:      r.Name,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	//
	h.vars.Extra[extraResult] = result.Result.Value

	//
	resp.Agent = r
	resp.Result = result.Result

	//
	h.vars.History = history
	return nil
}
