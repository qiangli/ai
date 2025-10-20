package swarm

import (
	"context"
	"maps"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// flow actions
func (h *agentHandler) action(ctx context.Context, tf api.ToolCondition) error {
	var r = h.agent

	// apply template/load
	// apply := func(vars *api.Vars, ext, s string) (string, error) {
	// 	//
	// 	if ext == "tpl" {
	// 		// TODO custom template func?
	// 		return applyTemplate(s, vars, tplFuncMap)
	// 	}
	// 	return s, nil
	// }

	// merge args
	var args = make(map[string]any)
	maps.Copy(args, h.globals)
	if r.Arguments != nil {
		maps.Copy(args, r.Arguments)
	}

	// // check agents in args
	// for key, val := range args {
	// 	if v, ok := val.(string); ok {
	// 		resolved, err := h.resolveArgument(ctx, req, v)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		args[key] = resolved
	// 	}
	// }

	args["query"] = h.agent.RawInput.Query()

	log.GetLogger(ctx).Debugf("Added user role message: %+v\n", args)

	// //
	// var runTool = h.createCaller()
	// runTool(ctx)

	// // h.vars.Extra[extraResult] = result.Result.Value
	// // h.vars.History = history

	// // resp.Agent = r
	// resp.Result = result.Result
	return nil
}

func (h *agentHandler) flowSequence(req *api.Request, resp *api.Response) error {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	// var shared = make(map[string]any)

	for _, action := range h.agent.Flow.Actions {
		if action.Tool.Type == api.ToolTypeAgent {
			req := api.NewRequest(ctx, action.Tool.Name, h.agent.RawInput.Clone())
			req.Parent = h.agent

			resp := &api.Response{}
			if err := h.exec(req, resp); err != nil {
				return err
			}
		}

		// args := make(map[string]any)
		// h.callTool(ctx, action.Tool, args)
	}
	return nil
}

func (h *agentHandler) flowParallel(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	return nil
}
