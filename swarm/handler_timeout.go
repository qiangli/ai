package swarm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func TimeoutMiddleware(agent *api.Agent) api.Middleware {
	mw := func(next Handler) Handler {
		th := &timeoutHandler{
			next:    next,
			content: fmt.Sprintf("%q timed out after %v seconds.", agent.Name, agent.MaxTime),
			dt:      time.Duration(agent.MaxTime) * time.Second,
		}

		return HandlerFunc(func(req *api.Request, res *api.Response) error {
			log.GetLogger(req.Context()).Debugf("🔗 (timeout): %s max_time: %v\n", agent.Name, agent.MaxTime)

			return th.Serve(req, res)
		})
	}

	return mw
}

// ErrHandlerTimeout is returned on [Response]
// in handlers which have timed out.
var ErrHandlerTimeout = errors.New("Agent service timeout")

type timeoutHandler struct {
	next    api.Handler
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
