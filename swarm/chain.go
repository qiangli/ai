// Adapted from https://github.com/justinas/alice/blob/master/chain.go
package swarm

import (
	"github.com/qiangli/ai/swarm/api"
)

type emptyHandler struct{}

func (h *emptyHandler) Serve(r *api.Request, w *api.Response) error {
	return nil
}

var emptyMiddleware = func(Handler) Handler {
	return HandlerFunc(func(*api.Request, *api.Response) error {
		return nil
	})
}

// A Handler responds to a request.
//
// [Handler.Serve] should write reply headers and data to the [ResponseWriter]
// and then return. Returning signals that the request is finished; it
// is not valid to use the [ResponseWriter] or read from the
// [Request.Body] after or concurrently with the completion of the
// Serve call.
//
// Depending on the client software, protocol version, and
// any intermediaries between the client and the Go server, it may not
// be possible to read from the [Request.Body] after writing to the
// [ResponseWriter]. Cautious handlers should read the [Request.Body]
// first, and then reply.
//
// Except for reading the body, handlers should not modify the
// provided Request.
//
// If Serve panics, the server (the caller of Serve) assumes
// that the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log,
// and either closes the network connection or sends an HTTP/2
// RST_STREAM, depending on the protocol. To abort a handler so
// the client sees an interrupted response but the server doesn't log
// an error, panic with the value [ErrAbortHandler].
// type Handler interface {
// 	Serve(*api.Request, *api.Response) error
// }

type Handler = api.Handler

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as handlers. If f is a function
// with the appropriate signature, HandlerFunc(f) is a
// [Handler] that calls f.
// type HandlerFunc func(*api.Request, *api.Response) error

type HandlerFunc = api.HandlerFunc

// // Serve calls f(w, r).
// func (f HandlerFunc) Serve(r *api.Request, w *api.Response) error {
// 	return f(r, w)
// }

// A constructor for a piece of middleware.
// Some middleware use this constructor out of the box,
// so in most cases you can just pass somepackage.New
// type Constructor func(api.Middleware) api.Middleware

// Chain acts as a list of Handler constructors.
// Chain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type Chain struct {
	constructors []api.Middleware
}

// New creates a new chain,
// memorizing the given list of middleware constructors.
// New serves no other function,
// constructors are only called upon a call to Then().
func NewChain(constructors ...api.Middleware) Chain {
	return Chain{append(([]api.Middleware)(nil), constructors...)}
}

// Then chains the middleware and returns the final Handler.
//
//	New(m1, m2, m3).Then(h)
//
// is equivalent to:
//
//	m1(m2(m3(h)))
//
// When the request comes in, it will be passed to m1, then m2, then m3
// and finally, the given handler
// (assuming every middleware calls the following one).
//
// A chain can be safely reused by calling Then() several times.
//
//	stdStack := alice.New(ratelimitHandler, csrfHandler)
//	indexPipe = stdStack.Then(indexHandler)
//	authPipe = stdStack.Then(authHandler)
//
// Note that constructors are called on every call to Then()
// and thus several instances of the same middleware will be created
// when a chain is reused in this way.
// For proper middleware, this should cause no problems.
//
// Then() treats nil as DefaultServeMux.
func (c Chain) Then(a *api.Agent, h api.Handler) api.Handler {
	if h == nil {
		// h = api.HandlerFunc(&emptyHandler{})
		// h = api.Middleware(*emptyHandler{})
		h = &emptyHandler{}
	}

	for i := range c.constructors {
		h = c.constructors[len(c.constructors)-1-i](a, h)
	}

	return h
}

// ThenFunc works identically to Then, but takes
// a HandlerFunc instead of a Handler.
//
// The following two statements are equivalent:
//
//	c.Then(HandlerFunc(fn))
//	c.ThenFunc(fn)
//
// ThenFunc provides all the guarantees of Then.
func (c Chain) ThenFunc(a *api.Agent, fn HandlerFunc) Handler {
	// This nil check cannot be removed due to the "nil is not nil" common mistake in Go.
	// Required due to: https://stackoverflow.com/questions/33426977/how-to-golang-check-a-variable-is-nil
	if fn == nil {
		return c.Then(a, nil)
	}
	return c.Then(a, fn)
}

// Append extends a chain, adding the specified constructors
// as the last ones in the request flow.
//
// Append returns a new chain, leaving the original one untouched.
//
//	stdChain := alice.New(m1, m2)
//	extChain := stdChain.Append(m3, m4)
//	// requests in stdChain go m1 -> m2
//	// requests in extChain go m1 -> m2 -> m3 -> m4
func (c Chain) Append(constructors ...api.Middleware) Chain {
	newCons := make([]api.Middleware, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)

	return Chain{newCons}
}

// Extend extends a chain by adding the specified chain
// as the last one in the request flow.
//
// Extend returns a new chain, leaving the original one untouched.
//
//	stdChain := alice.New(m1, m2)
//	ext1Chain := alice.New(m3, m4)
//	ext2Chain := stdChain.Extend(ext1Chain)
//	// requests in stdChain go  m1 -> m2
//	// requests in ext1Chain go m3 -> m4
//	// requests in ext2Chain go m1 -> m2 -> m3 -> m4
//
// Another example:
//
//	aHtmlAfterNosurf := alice.New(m2)
//	aHtml := alice.New(m1, func(h Handler) Handler {
//		csrf := nosurf.New(h)
//		csrf.SetFailureHandler(aHtmlAfterNosurf.ThenFunc(csrfFail))
//		return csrf
//	}).Extend(aHtmlAfterNosurf)
//	// requests to aHtml hitting nosurfs success handler go m1 -> nosurf -> m2 -> target-handler
//	// requests to aHtml hitting nosurfs failure handler go m1 -> nosurf -> m2 -> csrfFail
func (c Chain) Extend(chain Chain) Chain {
	return c.Append(chain.constructors...)
}
