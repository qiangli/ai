package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

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
	// if len(r.Messages) > 0 {
	// 	log.GetLogger(ctx).Debugf("%s %s\n", r.Messages[0].Role, clip(r.Messages[0].Content, h.max))
	// }

	err := h.next.Serve(r, w)

	log.GetLogger(ctx).Debugf("Response: %+v\n", w)
	if w.Messages != nil {
		for _, m := range w.Messages {
			log.GetLogger(ctx).Debugf("%s %s\n", m.Role, clip(m.Content, h.max))
		}
	}

	return err
}
