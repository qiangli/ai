package swarm

import (
	// "context"
	// "errors"
	// "time"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

// extra result key
// const extraResult = "result"

// // TODO
// type LLMAdapter func(context.Context, *llm.Request) (*llm.Response, error)

// var adapterRegistry map[string]LLMAdapter

// func init() {
// 	adapterRegistry = make(map[string]LLMAdapter)
// 	adapterRegistry["chat"] = Chat
// 	adapterRegistry["image-gen"] = ImageGen
// }

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
	ctx := r.Context()
	log.GetLogger(ctx).Debugf("Request: %+v\n", r)
	if len(r.Messages) > 0 {
		log.GetLogger(ctx).Debugf("%s %s\n", r.Messages[0].Role, clip(r.Messages[0].Content, h.max))
	}

	err := h.next.Serve(r, w)

	log.GetLogger(ctx).Debugf("Response: %+v\n", w)
	if w.Messages != nil {
		for _, m := range w.Messages {
			log.GetLogger(ctx).Debugf("%s %s\n", m.Role, clip(m.Content, h.max))
		}
	}

	return err
}

// TimeoutHandler returns a [Handler] that times out if the time limit is reached.
//
// The new Handler calls thext next handler's Serve to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a timeout error.
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
