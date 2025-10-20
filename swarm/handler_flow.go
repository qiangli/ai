package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// flow actions
func (h *agentHandler) doAction(ctx context.Context, req *api.Request, resp *api.Response, tf *api.ToolFunc) error {
	var r = h.agent

	// merge args
	var args = make(map[string]any)
	// defaults from agent first
	if r.Arguments != nil {
		maps.Copy(args, r.Arguments)
	}
	// copy globals
	maps.Copy(args, h.vars.Global)
	// check agents in args for string values
	for key, val := range args {
		if v, ok := val.(string); ok {
			resolved, err := h.resolveArgument(ctx, req, v)
			if err != nil {
				return err
			}
			args[key] = resolved
		}
	}
	// secode pass: templated args
	for key, val := range args {
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			resolved, err := applyTemplate(v, args, tplFuncMap)
			if err != nil {
				return err
			}
			args[key] = resolved
		}
	}

	log.GetLogger(ctx).Debugf("Added user role message: %+v\n", args)

	var runTool = h.createCaller()
	result, err := runTool(ctx, tf.ID(), args)
	if err != nil {
		return err
	}

	resp.Agent = r
	resp.Result = result
	// TODO check states?
	return nil
}

func (h *agentHandler) flowSequence(req *api.Request, resp *api.Response) error {
	ctx := req.Context()
	nreq := req.Clone()
	nresp := &api.Response{}
	for _, v := range h.agent.Flow.Actions {
		if err := h.doAction(ctx, nreq, nresp, v.Tool); err != nil {
			return err
		}
		nreq.RawInput = &api.UserInput{
			Message: resp.Result.Value,
		}
	}
	resp.Result = nresp.Result
	return nil
}

func (h *agentHandler) flowParallel(req *api.Request, resp *api.Response) error {
	return api.NewUnsupportedError("parallel")
}

func (h *agentHandler) flowLoop(req *api.Request, resp *api.Response) error {
	return api.NewUnsupportedError("loop")
}

func (h *agentHandler) flowNest(req *api.Request, resp *api.Response) error {
	return api.NewUnsupportedError("nest")
}

func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	var which int
	// evaluate express or random
	if h.agent.Flow.Expression != "" {
		v, err := applyTemplate(h.agent.Flow.Expression, h.vars.Global, tplFuncMap)
		if err != nil {
			return err
		}
		if v, err := strconv.ParseInt(v, 0, 64); err != nil {
			return err
		} else {
			which = int(v)
		}
	} else {
		which = rand.Intn(len(h.agent.Flow.Actions))
	}
	if which < 0 && which >= len(h.agent.Flow.Actions) {
		return fmt.Errorf("index out of bound; %v", which)
	}

	ctx := req.Context()

	v := h.agent.Flow.Actions[which]
	if err := h.doAction(ctx, req, resp, v.Tool); err != nil {
		return err
	}
	return nil
}

func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	result, ok := h.vars.Global[globalResult]
	if !ok {
		return fmt.Errorf("no result found")
	}
	var s string
	if v, ok := result.(string); ok {
		s = v
	} else {
		s = fmt.Sprintf("%v", v)
	}
	var list []string
	err := json.Unmarshal([]byte(s), &list)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var resps = make([]*api.Response, len(list))

	for i, v := range list {
		wg.Add(1)
		go func(i int, v string) {
			nreq := req.Clone()
			nreq.RawInput = &api.UserInput{
				Message: v,
			}
			nresp := new(api.Response)
			h.flowSequence(nreq, nresp)
			resps[i] = nresp
		}(i, v)
	}
	var results []string
	for _, v := range resps {
		results = append(results, v.Result.Value)
	}
	resp.Result = &api.Result{
		Value: strings.Join(results, "\n"),
	}
	return nil
}

func (h *agentHandler) flowReduce(req *api.Request, resp *api.Response) error {
	return api.NewUnsupportedError("reduce")
}
