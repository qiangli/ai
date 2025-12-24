package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
)

// type ActionRunner interface {
// 	Run(context.Context, string, map[string]any) (any, error)
// }

// TODO
// logging, analytics, and debugging.
// prompts, tool selection, and output formatting.
// retries, fallbacks, timeout, early termination.
// rate limits, guardrails, pii detection.

type ActionHandler interface {
	Serve(context.Context, *api.Vars, api.ArgMap) (any, error)
}

type ToolFuncActionHandler struct {
	next ActionHandler

	action string
	vars   *api.Vars
}

// Serve implements ActionHandler interface.
// Creates an alias linking back to this handler for calling the next handler.
func (r *ToolFuncActionHandler) Serve(ctx context.Context, vars *api.Vars, args api.ArgMap) (any, error) {
	// arbitrary alias name, prepending a prefix to avoid name conflicts
	var alias string
	kit, name := api.Kitname(r.action).Clean().Decode()
	if kit == "agent" {
		pack, sub := api.Packname(name).Clean().Decode()
		alias = fmt.Sprintf("chainlink_agent_%s_%s", pack, sub)
	} else {
		alias = fmt.Sprintf("chainlink_tool_%s_%s", kit, name)
	}

	// an alias action must be: alias:<name>
	// install the callback to the Run method
	args["action"] = fmt.Sprintf("alias:%s", alias)
	args[alias] = r

	// run action
	// the action in the chain must be one that takes
	// a sub-action, in this case, the alias action.
	result, err := vars.RootAgent.Runner.Run(ctx, r.action, args)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Run implements ActionRunner interface
func (r *ToolFuncActionHandler) Run(ctx context.Context, _ string, args map[string]any) (any, error) {
	return r.next.Serve(ctx, r.vars, args)
}

func NewToolFuncActionMiddleware(vars *api.Vars, action string) func(ActionHandler) ActionHandler {
	return func(ah ActionHandler) ActionHandler {
		return &ToolFuncActionHandler{
			next:   ah,
			action: action,
			vars:   vars,
		}
	}
}

// The ActionHandlerFunc type is an adapter to allow the use of
// ordinary functions as action handlers.
type ActionHandlerFunc func(ctx context.Context, vars *api.Vars, argm api.ArgMap) (any, error)

func (f ActionHandlerFunc) Serve(ctx context.Context, vars *api.Vars, argm api.ArgMap) (any, error) {
	return f(ctx, vars, argm)
}

var defaultActionHandler = ActionHandlerFunc(func(ctx context.Context, vars *api.Vars, argm api.ArgMap) (any, error) {
	return nil, nil
})

func StartChainActions(ctx context.Context, vars *api.Vars, actions []string, args api.ArgMap) (any, error) {
	final := ActionHandlerFunc(func(ctx context.Context, vars *api.Vars, args api.ArgMap) (any, error) {
		return nil, nil
	})

	var middlewares []ActionMiddleware
	for _, v := range actions {
		mw := NewToolFuncActionMiddleware(vars, v)
		middlewares = append(middlewares, mw)
	}

	chain := New(middlewares...).Then(final)
	result, err := chain.Serve(ctx, vars, args)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// https://github.com/justinas/alice/blob/master/chain.go
type ActionMiddleware func(ActionHandler) ActionHandler

// ActionChain acts as a list of ActionHandler constructors.
// ActionChain is effectively immutable:
// once created, it will always hold
// the same set of constructors in the same order.
type ActionChain struct {
	constructors []ActionMiddleware
}

// New creates a new chain,
// memorizing the given list of middleware constructors.
// New serves no other function,
// constructors are only called upon a call to Then().
func New(constructors ...ActionMiddleware) ActionChain {
	return ActionChain{append(([]ActionMiddleware)(nil), constructors...)}
}

// type ActionMiddleware Constructor

// Then chains the middleware and returns the final ActionHandler.
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
// Then() treats nil as http.DefaultServeMux.
func (c ActionChain) Then(h ActionHandler) ActionHandler {
	if h == nil {
		h = defaultActionHandler
	}

	for i := range c.constructors {
		h = c.constructors[len(c.constructors)-1-i](h)
	}
	return h
}

// ThenFunc works identically to Then, but takes
// a HandlerFunc instead of a Handler.
//
// The following two statements are equivalent:
//
//	c.Then(http.HandlerFunc(fn))
//	c.ThenFunc(fn)
//
// ThenFunc provides all the guarantees of Then.
func (c ActionChain) ThenFunc(fn ActionHandlerFunc) ActionHandler {
	// This nil check cannot be removed due to the "nil is not nil" common mistake in Go.
	// Required due to: https://stackoverflow.com/questions/33426977/how-to-golang-check-a-variable-is-nil
	if fn == nil {
		return c.Then(nil)
	}
	return c.Then(fn)
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
func (c ActionChain) Append(constructors ...ActionMiddleware) ActionChain {
	newCons := make([]ActionMiddleware, 0, len(c.constructors)+len(constructors))
	newCons = append(newCons, c.constructors...)
	newCons = append(newCons, constructors...)

	return ActionChain{newCons}
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
//	aHtml := alice.New(m1, func(h ActionHandler) ActionHandler {
//		csrf := nosurf.New(h)
//		csrf.SetFailureHandler(aHtmlAfterNosurf.ThenFunc(csrfFail))
//		return csrf
//	}).Extend(aHtmlAfterNosurf)
//	// requests to aHtml hitting nosurfs success handler go m1 -> nosurf -> m2 -> target-handler
//	// requests to aHtml hitting nosurfs failure handler go m1 -> nosurf -> m2 -> csrfFail
func (c ActionChain) Extend(chain ActionChain) ActionChain {
	return c.Append(chain.constructors...)
}
