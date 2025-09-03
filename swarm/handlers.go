package swarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
)

// extra result key
const extraResult = "result"

// TODO
type LLMAdapter func(context.Context, *api.LLMRequest) (*api.LLMResponse, error)

var adapterRegistry map[string]LLMAdapter

func init() {
	adapterRegistry = make(map[string]LLMAdapter)
	adapterRegistry["chat"] = llm.Chat
	adapterRegistry["image-gen"] = llm.ImageGen
}

// AgentHandler
func AgentHandler(vars *api.Vars, agent *api.Agent) Handler {
	return &agentHandler{
		vars:  vars,
		agent: agent,
	}
}

type agentHandler struct {
	agent *api.Agent
	vars  *api.Vars
}

func (h *agentHandler) Serve(req *api.Request, resp *api.Response) error {
	var r = h.agent
	log.Debugf("run agent: %s\n", r.Name)

	ctx := req.Context()

	// dependencies
	if len(r.Dependencies) > 0 {
		for _, agent := range r.Dependencies {
			depReq := &api.Request{
				Agent:    agent,
				RawInput: req.RawInput,
				Messages: req.Messages,
			}
			depResp := &api.Response{}
			sw := New(h.vars)
			if err := sw.Run(depReq, depResp); err != nil {
				return err
			}
			//

			// decode prevous result
			// decode content as name=value and save in vars.Extra for subsequent agents
			if v, ok := h.vars.Extra[extraResult]; ok && len(v) > 0 {
				var params = make(map[string]string)
				if err := json.Unmarshal([]byte(v), &params); err == nil {
					maps.Copy(h.vars.Extra, params)
				}
			}

			log.Debugf("run dependency: %s %+v\n", agent, depResp)
		}
	}

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
	apply := func(ext, s string, vars *api.Vars) (string, error) {
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
	if r.Config != nil && r.Config.Instruction != nil {
		// update the request instruction
		instruction := r.Config.Instruction
		content, err := apply(instruction.Type, instruction.Content, h.vars)
		if err != nil {
			return err
		}

		if log.IsTrace() {
			log.Debugf("content: %s\n", content)
		}

		history = append(history, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    nvl(instruction.Role, api.RoleSystem),
			Content: content,
			Sender:  r.Name,
			Models:  h.vars.Config.Models,
		})
		log.Debugf("Added new system role message: %v\n", len(history))
	}

	// 2. Historical Messages - skip system role
	if len(h.vars.History) > 0 {
		log.Debugf("using %v messaages from history\n", len(h.vars.History))
		for _, msg := range h.vars.History {
			if msg.Role != api.RoleSystem {
				history = append(history, msg)
				log.Debugf("Added historical non system role message: %v\n", len(history))
			}
		}
	}

	// 3. New User Message
	if req.Messages == nil {
		// contentType/content
		messages, err := req.RawInput.FileMessages()
		if err != nil {
			return err
		}
		for i, v := range messages {
			v.ID = uuid.NewString()
			v.ChatID = chatID
			v.Created = time.Now()
			//
			v.Role = api.RoleUser
			v.Sender = r.Name
			v.Models = h.vars.Config.Models
			messages[i] = v
		}

		req.Messages = messages
		req.Messages = append(req.Messages, &api.Message{
			ID:      uuid.NewString(),
			ChatID:  chatID,
			Created: time.Now(),
			//
			Role:    api.RoleUser,
			Content: req.RawInput.Query(),
			Sender:  r.Name,
			Models:  h.vars.Config.Models,
		})
	}
	history = append(history, req.Messages...)
	log.Debugf("Added new user role message: %v\n", len(history))

	// Request
	initLen := len(history)

	runTool := func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		return CallTool(ctx, h.vars, name, args)
	}

	// send message to LLM
	model, err := h.vars.Config.ModelLoader(r.Model)
	if err != nil {
		return fmt.Errorf("failed to load model %q: %v", r.Model, err)
	}

	var request = api.LLMRequest{
		Agent:    r.Name,
		Model:    model,
		Messages: history,
		MaxTurns: r.MaxTurns,
		RunTool:  runTool,
		Tools:    r.Tools,
		//
		Vars: h.vars,
	}

	if log.IsTrace() {
		log.Debugf("LLM request: %+v\n", request)
	}

	var adapter LLMAdapter = llm.Chat
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

	if log.IsTrace() {
		log.Debugf("LLM response: %+v\n", result)
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
			Models:      h.vars.Config.Models,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	if log.IsTrace() {
		log.Debugf("Response messages: %+v", resp.Messages)
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

// MaxLogHandler returns a [Handler] that logs the request and response
func MaxLogHandler(n int) func(Handler) Handler {
	return func(next Handler) Handler {
		return &maxLogHandler{
			next: next,
			max:  n,
		}
	}
}

type maxLogHandler struct {
	next Handler
	max  int
}

func (h *maxLogHandler) Serve(r *api.Request, w *api.Response) error {

	log.Debugf("req: %+v\n", r)
	if len(r.Messages) > 0 {
		log.Debugf("%s %s\n", r.Messages[0].Role, clip(r.Messages[0].Content, h.max))
	}

	err := h.next.Serve(r, w)

	log.Debugf("resp: %+v\n", w)
	if w.Messages != nil {
		for _, m := range w.Messages {
			log.Debugf("%s %s\n", m.Role, clip(m.Content, h.max))
		}
	}

	return err
}

// TimeoutHandler returns a [Handler] that times out if the time limit is reached.
//
// The new Handler calls thext next handler's Serve to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a timeout error.
func TimeoutHandler(next Handler, dt time.Duration, msg string) Handler {
	return &timeoutHandler{
		next:    next,
		content: msg,
		dt:      dt,
	}
}

// ErrHandlerTimeout is returned on [Response]
// in handlers which have timed out.
var ErrHandlerTimeout = errors.New("Agent service timeout")

type timeoutHandler struct {
	next    Handler
	content string
	dt      time.Duration
}

func (h *timeoutHandler) Serve(r *api.Request, w *api.Response) error {
	ctx, cancelCtx := context.WithTimeout(r.Context(), h.dt)
	defer cancelCtx()

	r = r.WithContext(ctx)

	done := make(chan struct{})
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()

		if err := h.next.Serve(r, w); err != nil {
			panicChan <- err
		}

		close(done)
	}()

	select {
	case p := <-panicChan:
		return p.(error)
	case <-done:
		return nil
	case <-ctx.Done():
		w.Messages = []*api.Message{{Content: h.content}}
		w.Agent = nil
	}

	return ErrHandlerTimeout
}
