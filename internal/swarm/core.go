package swarm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qiangli/ai/internal/log"
)

type Swarm struct {
	History []*Message
	Vars    *Vars
	Stream  bool

	//
	Config *AgentsConfig
	Create AgentFunc

	//
	DryRun        bool
	DryRunContent string
}

func NewSwarm(config *AgentsConfig) *Swarm {
	return &Swarm{
		Vars:    NewVars(),
		History: []*Message{},
		Stream:  true,
		Config:  config,
		Create:  AgentCreator(config),
	}
}

func (r *Swarm) Run(req *Request, resp *Response) error {
	resourceMap := r.Config.ResourceMap

	// "resource:" prefix is used to refer to a resource
	// "vars:" prefix is used to refer to a variable
	apply := func(s string, vars *Vars) (string, error) {
		if strings.HasPrefix(s, "resource:") {
			v, ok := resourceMap[s[9:]]
			if !ok {
				return "", fmt.Errorf("no such resource: %s", s[9:])
			}
			return applyTemplate(v, vars, r.Config.TemplateFuncMap)
		}
		if strings.HasPrefix(s, "vars:") {
			v := vars.GetString(s[5:])
			return v, nil
		}
		return s, nil
	}

	for {
		agent, err := r.Create(req.Agent, r.Vars)
		if err != nil {
			return err
		}
		agent.sw = r

		if agent.Entrypoint != nil {
			if err := agent.Entrypoint(r.Vars, agent, req.RawInput); err != nil {
				return err
			}
		}

		// update the request instruction
		content, err := apply(agent.Instruction, r.Vars)
		if err != nil {
			return err
		}
		agent.Instruction = content

		timeout := TimeoutHandler(agent, time.Duration(agent.MaxTime)*time.Second, "timed out")
		maxlog := MaxLogHandler(500)

		chain := New(maxlog).Then(timeout)

		if err := chain.Serve(req, resp); err != nil {
			return err
		}

		// update the request
		if resp.Transfer {
			req.Agent = resp.NextAgent
			continue
		}
		return nil
	}
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

func (h *maxLogHandler) Serve(r *Request, w *Response) error {

	log.Debugf("req: %+v\n", r)
	if r.Message != nil {
		log.Printf("%s %s\n", r.Message.Role, clip(r.Message.Content, h.max))
	}

	err := h.next.Serve(r, w)

	log.Debugf("resp: %+v\n", w)
	if w.Messages != nil {
		for _, m := range w.Messages {
			log.Printf("%s %s\n", m.Role, clip(m.Content, h.max))
		}
	}

	return err
}

// TimeoutHandler returns a [Handler] that runs h with the given time limit.
//
// The new Handler calls h.Serve to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a timeout error.
// After such a timeout, writes by h to its [Response] will return
// [ErrHandlerTimeout].
func TimeoutHandler(next Handler, dt time.Duration, msg string) Handler {
	return &timeoutHandler{
		next:    next,
		content: msg,
		dt:      dt,
	}
}

// ErrHandlerTimeout is returned on [Response]
// in handlers which have timed out.
var ErrHandlerTimeout = errors.New("Handler timeout")

type timeoutHandler struct {
	next    Handler
	content string
	dt      time.Duration
}

func (h *timeoutHandler) Serve(r *Request, w *Response) error {
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

		h.next.Serve(r, w)

		close(done)
	}()

	select {
	case p := <-panicChan:
		return p.(error)
	case <-done:
		return nil
	case <-ctx.Done():
		w.Messages = []*Message{{Content: h.content}}
		w.Agent = nil
	}

	return ErrHandlerTimeout
}
