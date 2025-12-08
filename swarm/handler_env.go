package swarm

import (
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/log"
)

// Init and start the chain
func InitEnvMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())

			var ctx = req.Context()

			// update envs and args
			//
			// envs
			var envs = make(map[string]any)
			maps.Copy(envs, sw.globalEnv())

			//
			// inherit envs of embeded agents
			// merge into args if not already set in args
			add := func(e *api.Environment) error {
				return sw.mapAssign(ctx, agent, envs, e.GetAllEnvs(), true)
			}

			var addAll func(*api.Agent) error
			addAll = func(a *api.Agent) error {
				for _, v := range a.Embed {
					if err := addAll(v); err != nil {
						return err
					}
				}
				if a.Environment != nil {
					if err := add(a.Environment); err != nil {
						return err
					}
				}
				return nil
			}

			addAll(agent)
			agent.Environment.AddEnvs(envs)

			// args
			//
			// global/agent envs
			// agent args
			// req args
			var args = make(map[string]any)
			maps.Copy(args, envs)
			if agent.Arguments != nil {
				aargs := agent.Arguments.GetAllArgs()
				if err := sw.mapAssign(ctx, agent, args, aargs, true); err != nil {
					return err
				}
			}
			if req.Arguments != nil {
				rargs := req.Arguments.GetAllArgs()
				if err := sw.mapAssign(ctx, agent, args, rargs, true); err != nil {
					return err
				}
			}

			req.Arguments.SetArgs(args)

			ll := req.Arguments.GetString("log_level")
			logger.SetLogLevel(api.ToLogLevel(ll))

			var parent string
			if agent.Parent != nil {
				parent = fmt.Sprintf("%s → ", agent.Parent.Name)
			}
			logger.Infof("🚀 %s%s\n", parent, agent.Name)

			return next.Serve(req, resp)
		})
	}
}

// Apply template and update agent enviroment
func ApplyEnv(global map[string]any, agent *api.Agent, args map[string]any) error {
	// envs
	var envs = make(map[string]any)
	maps.Copy(envs, global)

	//
	// inherit envs of embeded agents
	// merge into args if not already set in args
	add := func(e *api.Environment) error {
		return mapAssign(agent.Template, envs, e.GetAllEnvs())
	}

	var addAll func(*api.Agent) error
	addAll = func(a *api.Agent) error {
		for _, v := range a.Embed {
			if err := addAll(v); err != nil {
				return err
			}
		}
		if a.Environment != nil {
			if err := add(a.Environment); err != nil {
				return err
			}
		}
		return nil
	}

	addAll(agent)
	agent.Environment.AddEnvs(envs)

	// args
	//
	// global
	// agent envs
	// agent args
	var nargs = make(map[string]any)
	maps.Copy(nargs, envs)
	if agent.Arguments != nil {
		// aargs := agent.Arguments.GetAllArgs()
		if err := mapAssign(agent.Template, nargs, agent.Arguments.GetAllArgs()); err != nil {
			return err
		}
	}

	if err := mapAssign(agent.Template, nargs, args); err != nil {
		return err
	}

	return nil
}

func mapAssign(tpl *template.Template, dst, src map[string]any) error {
	for key, val := range src {
		// go template value support
		if v, ok := val.(string); ok && strings.Contains(v, "{{") {
			if resolved, err := atm.ApplyTemplate(tpl, v, dst); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		dst[key] = val
	}
	return nil
}
