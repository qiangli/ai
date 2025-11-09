package swarm

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// MaxLogMiddleware returns a [api.Middleware] that logs the request and response
func MaxLogMiddleware(n int) api.Middleware {
	mw := func(next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			ctx := req.Context()
			log.GetLogger(ctx).Debugf("Request: %+v\n", req)

			err := next.Serve(req, resp)

			log.GetLogger(ctx).Debugf("Response: %+v\n", resp)
			if resp.Messages != nil {
				for _, m := range resp.Messages {
					log.GetLogger(ctx).Debugf("%s %s\n", m.Role, clip(m.Content, n))
				}
			}

			return err
		})
	}
	return mw
}

// type maxLogHandler struct {
// 	next api.Handler
// 	max  int
// }

// func (h *maxLogHandler) Serve(r *api.Request, w *api.Response) error {
// 	ctx := r.Context()
// 	log.GetLogger(ctx).Debugf("Request: %+v\n", r)

// 	err := h.next.Serve(r, w)

// 	log.GetLogger(ctx).Debugf("Response: %+v\n", w)
// 	if w.Messages != nil {
// 		for _, m := range w.Messages {
// 			log.GetLogger(ctx).Debugf("%s %s\n", m.Role, clip(m.Content, h.max))
// 		}
// 	}

// 	return err
// }
