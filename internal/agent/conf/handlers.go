package conf

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

// extra result key
const extraResult = "result"

// TODO
type LLMAdapter func(context.Context, *llm.Request) (*llm.Response, error)

var adapterRegistry map[string]LLMAdapter

func init() {
	adapterRegistry = make(map[string]LLMAdapter)
	adapterRegistry["chat"] = swarm.Chat
	adapterRegistry["image-gen"] = swarm.ImageGen
}

func NewAgentHandler() func(*api.Vars, *api.Agent) api.Handler {
	var toolCall = NewToolCaller()
	return func(vars *api.Vars, agent *api.Agent) api.Handler {
		return &agentHandler{
			vars:     vars,
			agent:    agent,
			toolCall: toolCall,
		}
	}
}

type agentHandler struct {
	agent *api.Agent
	vars  *api.Vars

	toolCall api.ToolCaller
}

func (h *agentHandler) Serve(req *api.Request, resp *api.Response) error {
	var r = h.agent
	var ctx = req.Context()
	log.GetLogger(ctx).Debugf("Run agent: %s\n", r.Name)

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

	nvl := func(a string, b ...string) string {
		if a != "" {
			return a
		}
		for _, v := range b {
			if v != "" {
				return v
			}
		}
		return ""
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

		if log.GetLogger(ctx).IsTrace() {
			log.GetLogger(ctx).Debugf("Content: %s\n", content)
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
	if !r.New && len(h.vars.History) > 0 {
		log.GetLogger(ctx).Debugf("Adding messaages from history\n")
		for _, msg := range h.vars.History {
			if msg.Role != api.RoleSystem {
				history = append(history, msg)
				log.GetLogger(ctx).Debugf("Added %q message\n", msg.Role)
			}
		}
		log.GetLogger(ctx).Debugf("Added %v messages\n", len(history))
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
	// user message
	var content = req.RawInput.Query()
	if r.Message != "" {
		content = r.Message + "\n" + content
	}
	req.Messages = append(req.Messages, &api.Message{
		ID:      uuid.NewString(),
		ChatID:  chatID,
		Created: time.Now(),
		//
		Role:    api.RoleUser,
		Content: content,
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

	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("LLM request: %+v\n", request)
	}

	var adapter LLMAdapter = swarm.Chat
	if h.agent.Adapter != "" {
		if v, ok := adapterRegistry[h.agent.Adapter]; ok {
			adapter = v
		} else {
			return fmt.Errorf("LLM adapter not found: %v", h.agent.Adapter)
		}
	}

	result, err := adapter(ctx, &request)
	if err != nil {
		return err
	}

	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("LLM response: %+v\n", result)
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

	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("Response messages: %+v", resp.Messages)
	}

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

// // MaxLogHandler returns a [Handler] that logs the request and response
// func MaxLogHandler(n int) func(Handler) Handler {
// 	return func(next Handler) Handler {
// 		return &maxLogHandler{
// 			next: next,
// 			max:  n,
// 		}
// 	}
// }

// type maxLogHandler struct {
// 	next Handler
// 	max  int
// }

// func (h *maxLogHandler) Serve(r *api.Request, w *api.Response) error {
// 	ctx := r.Context()
// 	log.GetLogger(ctx).Debugf("Request: %+v\n", r)
// 	if len(r.Messages) > 0 {
// 		log.GetLogger(ctx).Debugf("%s %s\n", r.Messages[0].Role, clip(r.Messages[0].Content, h.max))
// 	}

// 	err := h.next.Serve(r, w)

// 	log.GetLogger(ctx).Debugf("Response: %+v\n", w)
// 	if w.Messages != nil {
// 		for _, m := range w.Messages {
// 			log.GetLogger(ctx).Debugf("%s %s\n", m.Role, clip(m.Content, h.max))
// 		}
// 	}

// 	return err
// }

// // TimeoutHandler returns a [Handler] that times out if the time limit is reached.
// //
// // The new Handler calls thext next handler's Serve to handle each request, but if a
// // call runs for longer than its time limit, the handler responds with
// // a timeout error.
// func TimeoutHandler(next Handler, dt time.Duration, msg string) Handler {
// 	return &timeoutHandler{
// 		next:    next,
// 		content: msg,
// 		dt:      dt,
// 	}
// }

// // ErrHandlerTimeout is returned on [Response]
// // in handlers which have timed out.
// var ErrHandlerTimeout = errors.New("Agent service timeout")

// type timeoutHandler struct {
// 	next    Handler
// 	content string
// 	dt      time.Duration
// }

// func (h *timeoutHandler) Serve(r *api.Request, w *api.Response) error {
// 	ctx, cancelCtx := context.WithTimeout(r.Context(), h.dt)
// 	defer cancelCtx()

// 	r = r.WithContext(ctx)

// 	done := make(chan struct{})
// 	panicChan := make(chan any, 1)

// 	go func() {
// 		defer func() {
// 			if p := recover(); p != nil {
// 				panicChan <- p
// 			}
// 		}()

// 		if err := h.next.Serve(r, w); err != nil {
// 			panicChan <- err
// 		}

// 		close(done)
// 	}()

// 	select {
// 	case p := <-panicChan:
// 		return p.(error)
// 	case <-done:
// 		return nil
// 	case <-ctx.Done():
// 		w.Messages = []*api.Message{{Content: h.content}}
// 		w.Agent = nil
// 	}

// 	return ErrHandlerTimeout
// }
