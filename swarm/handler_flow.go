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
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/shell/tool/sh"
)

func (h *agentHandler) doAction(ctx context.Context, req *api.Request, resp *api.Response, action *api.Action) error {
	var args = make(map[string]any)
	if req.Arguments != nil {
		req.Arguments.Copy(args)
	}
	result, err := h.agent.Runner.Run(ctx, action.ID, args)
	resp.Agent = h.agent
	resp.Result = api.ToResult(result)
	return err
}

// FlowTypeSequence executes actions one after another, where each
// subsequent action uses the previous action's response as input.
func (h *agentHandler) flowSequence(req *api.Request, resp *api.Response) error {
	ctx := req.Context()
	nreq := req.Clone()
	nresp := &api.Response{}
	for _, v := range h.agent.Flow.Actions {
		if err := h.doAction(ctx, nreq, nresp, v); err != nil {
			return err
		}
		nreq.SetMessage(nresp.Result.Value)
	}

	// final result
	resp.Result = nresp.Result
	return nil
}

// FlowTypeLoop executes actions repetitively in a loop. The loop can use a counter or
// evaluate an expression for each iteration, allowing for repeated execution with varying
// parameters or conditions.
func (h *agentHandler) flowLoop(req *api.Request, resp *api.Response) error {
	env := h.sw.globalEnv()
	// h.mapAssign(req, env, req.Arguments, false)

	eval := func(exp string) (bool, error) {
		v, err := atm.ApplyTemplate(h.agent.Template, exp, env)
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
			// use the same request and respone
			if err := h.flowSequence(req, resp); err != nil {
				return err
			}
		}
	}
}

// FlowTypeParallel executes actions simultaneously, returning the combined results as a list.
// This allows for concurrent processing of independent actions.
func (h *agentHandler) flowParallel(req *api.Request, resp *api.Response) error {
	var ctx = req.Context()
	var resps = make([]*api.Response, len(h.agent.Flow.Actions))

	var wg sync.WaitGroup
	for i, v := range h.agent.Flow.Actions {
		wg.Add(1)
		go func(i int, v *api.Action) {
			defer wg.Done()

			// use the same request
			nresp := new(api.Response)
			if err := h.doAction(ctx, req, nresp, v); err != nil {
				nresp.Result = &api.Result{
					Value: err.Error(),
				}
			}
			resps[i] = nresp
		}(i, v)
	}
	wg.Wait()

	resp.Result = &api.Result{
		Value: marshalResponseList(resps),
	}
	return nil
}

// FlowTypeChoice selects and executes a single action based on an evaluated expression.
// If no expression is provided, an action is chosen randomly. The expression must evaluate
// to a string (tool ID), false/true, or an integer that selects the action index, starting from zero.
func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	env := h.sw.globalEnv()
	// h.mapAssign(req, env, req.Arguments, false)

	var which int = -1
	// evaluate express or random
	if h.agent.Flow.Expression != "" {
		v, err := atm.ApplyTemplate(h.agent.Template, h.agent.Flow.Expression, env)
		if err != nil {
			return err
		}
		// match the action id
		id := api.KitName(v).ID()
		for i, action := range h.agent.Flow.Actions {
			if id == action.ID {
				which = i
			}
		}
		//
		if b, err := strconv.ParseBool(v); err == nil {
			if b {
				which = 1
			} else {
				which = 0
			}
		}
		if which < 0 {
			if v, err := strconv.ParseInt(v, 0, 64); err != nil {
				return err
			} else {
				which = int(v)
			}
		}
	} else {
		// random
		which = rand.Intn(len(h.agent.Flow.Actions))
	}
	if which < 0 && which >= len(h.agent.Flow.Actions) {
		return fmt.Errorf("index out of bound; %v", which)
	}

	ctx := req.Context()

	v := h.agent.Flow.Actions[which]
	if err := h.doAction(ctx, req, resp, v); err != nil {
		return err
	}
	return nil
}

// FlowTypeMap applies specified action(s) to each element in the input array, creating a new
// array populated with the results.
func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	// if the map flow is the first in the pipeline
	// use query
	result, ok := h.sw.Vars.Global.Get(globalResult)
	if !ok {
		// result, _ = h.sw.Vars.Global.Get(globalQuery)
		result = req.Message()
	}

	tasks := unmarshalResultList(result)

	var resps = make([]*api.Response, len(tasks))

	var wg sync.WaitGroup
	for i, v := range tasks {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()

			nreq := req.Clone()
			nreq.SetMessage(v)
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

	resp.Result = &api.Result{
		Value: marshalResponseList(resps),
	}
	return nil
}

// FlowTypeShell delegates control to a shell script using bash script syntax, enabling
// complex flow control scenarios driven by external scripting logic.
func (h *agentHandler) flowShell(req *api.Request, resp *api.Response) error {
	ctx := req.Context()

	runner := NewAgentScriptRunner(h.sw, h.agent)

	// make a copy of the args which already include args from the agent
	var args = make(map[string]any)
	if req.Arguments != nil {
		req.Arguments.Copy(args)
	}

	data, err := runner.Run(ctx, h.agent.Flow.Script, args)
	if err != nil {
		return err
	}

	result := api.ToResult(data)

	resp.Result = &api.Result{
		Value: result.Value,
	}
	return nil
}

func doBashCustom(vs *sh.VirtualSystem, args []string) (string, error) {
	switch args[0] {
	case "env", "printenv":
		for k, v := range vs.System.Environ() {
			fmt.Fprintf(vs.IOE.Stdout, "%s=%v\n", k, v)
		}
	default:
	}
	return "", nil
}

// Unmarshal the result into a list.
// If the result isn't a list, return the result as a single-item list.
func unmarshalResultList(result any) []string {
	var s string
	if v, ok := result.(string); ok {
		s = v
	} else {
		s = fmt.Sprintf("%v", v)
	}
	var list []string
	err := json.Unmarshal([]byte(s), &list)
	if err != nil {
		list = []string{s}
	}
	return list
}

func marshalResponseList(resps []*api.Response) string {
	var results []string
	for _, v := range resps {
		results = append(results, v.Result.Value)
	}
	b, err := json.Marshal(results)
	if err != nil {
		return strings.Join(results, " ")
	}
	return string(b)
}
