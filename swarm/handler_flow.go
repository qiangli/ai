package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"github.com/qiangli/ai/swarm/api"
)

// flow actions
func (h *agentHandler) doAction(ctx context.Context, req *api.Request, resp *api.Response, tf *api.ToolFunc) error {
	var r = h.agent

	args, err := h.applyArguments(req)
	if err != nil {
		return err
	}

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
			Message: nresp.Result.Value,
		}
	}
	resp.Result = nresp.Result
	return nil
}

func (h *agentHandler) flowLoop(req *api.Request, resp *api.Response) error {
	eval := func(exp string) (bool, error) {
		v, err := applyTemplate(exp, h.sw.Vars.Global, tplFuncMap)
		if err != nil {
			return false, err
		}
		return strconv.ParseBool(v)
	}

	for {
		ok, err := eval(h.agent.Flow.Expression)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		if ok {
			if err := h.flowSequence(req, resp); err != nil {
				return err
			}
		}
	}
}

func (h *agentHandler) flowParallel(req *api.Request, resp *api.Response) error {
	var wg sync.WaitGroup

	var resps = make([]*api.Response, len(h.agent.Flow.Actions))
	var ctx = req.Context()
	for i, v := range h.agent.Flow.Actions {
		wg.Add(1)
		go func(i int, v *api.Action) {
			defer wg.Done()

			nresp := new(api.Response)
			if err := h.doAction(ctx, req, nresp, v.Tool); err != nil {
				nresp.Result = &api.Result{
					Value: err.Error(),
				}
			}
			resps[i] = nresp
		}(i, v)
	}
	wg.Wait()

	var results []string
	for _, v := range resps {
		results = append(results, v.Result.Value)
	}
	resp.Result = &api.Result{
		Value: strings.Join(results, "\n"),
	}
	return nil
}

func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	var which int
	// evaluate express or random
	if h.agent.Flow.Expression != "" {
		v, err := applyTemplate(h.agent.Flow.Expression, h.sw.Vars.Global, tplFuncMap)
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
	result, ok := h.sw.Vars.Global[globalResult]
	if !ok {
		// return fmt.Errorf("no result found")
		result = h.sw.Vars.Global[globalQuery]
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
		// return err
		list = []string{s}
	}

	var wg sync.WaitGroup
	var resps = make([]*api.Response, len(list))
	for i, v := range list {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()

			nreq := req.Clone()
			nreq.RawInput = &api.UserInput{
				Message: v,
			}
			nresp := new(api.Response)
			if err := h.flowSequence(nreq, nresp); err != nil {
				nresp.Result = &api.Result{
					Value: err.Error(),
				}
			}
			resps[i] = nresp
		}(i, v)
	}
	wg.Wait()

	var results []string
	for _, v := range resps {
		results = append(results, v.Result.Value)
	}

	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	resp.Result = &api.Result{
		Value: string(b),
	}
	return nil
}

func (h *agentHandler) flowReduce(req *api.Request, resp *api.Response) error {
	return api.NewUnsupportedError("reduce")
}

func (h *agentHandler) flowNest(req *api.Request, resp *api.Response) error {
	return api.NewUnsupportedError("nest")
}
