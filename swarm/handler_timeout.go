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

func (h *timeoutHandler) Serve(req *api.Request, resp *api.Response) error {
	ctx, cancelCtx := context.WithTimeout(req.Context(), h.dt)
	defer cancelCtx()

	nreq := req.WithContext(ctx)

	done := make(chan struct{})
	panicChan := make(chan any, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()

		if err := h.next.Serve(nreq, resp); err != nil {
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
		resp.Messages = []*api.Message{{Content: h.content}}
		resp.Agent = nil
	}

	return ErrHandlerTimeout
}
