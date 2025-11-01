package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/interp"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
)

// flow actions
func (h *agentHandler) doAction(ctx context.Context, req *api.Request, resp *api.Response, tf *api.ToolFunc) error {
	var r = h.agent

	args, err := h.applyArguments(req)
	if err != nil {
		return err
	}

	var runTool = h.createCaller(h.sw.User)
	result, err := runTool(ctx, tf.ID(), args)
	if err != nil {
		return err
	}

	resp.Agent = r
	resp.Result = result
	// TODO check states?
	return nil
}

// FlowTypeSequence executes actions one after another, where each
// subsequent action uses the previous action's response as input.
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

	// final result
	resp.Result = nresp.Result
	return nil
}

// FlowTypeLoop executes actions repetitively in a loop. The loop can use a counter or
// evaluate an expression for each iteration, allowing for repeated execution with varying
// parameters or conditions.
func (h *agentHandler) flowLoop(req *api.Request, resp *api.Response) error {
	eval := func(exp string) (bool, error) {
		v, err := h.applyTemplate(exp, h.sw.Vars.Global)
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
			if err := h.doAction(ctx, req, nresp, v.Tool); err != nil {
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
// to an integer that selects the action index, starting from zero.
func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	var which int
	// evaluate express or random
	if h.agent.Flow.Expression != "" {
		v, err := h.applyTemplate(h.agent.Flow.Expression, h.sw.Vars.Global)
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

// FlowTypeMap applies specified action(s) to each element in the input array, creating a new
// array populated with the results.
func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	// if the map flow is the first in the pipeline
	// use query
	result, ok := h.sw.Vars.Global.Get(globalResult)
	if !ok {
		result, _ = h.sw.Vars.Global.Get(globalQuery)
	}

	tasks := unmarshalResultList(result)

	var resps = make([]*api.Response, len(tasks))

	var wg sync.WaitGroup
	for i, v := range tasks {
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

	resp.Result = &api.Result{
		Value: marshalResponseList(resps),
	}
	return nil
}

// FlowTypeReduce applies action(s) sequentially to each element of an input array, accumulating
// results. It passes the result of each action as input to the next. The process returns a single
// accumulated value. If at the root, an initial value is sourced from a previous agent or user query.
func (h *agentHandler) flowReduce(req *api.Request, resp *api.Response) error {
	// if the map flow is the first in the pipeline
	// use query
	result, ok := h.sw.Vars.Global.Get(globalResult)
	if !ok {
		result, _ = h.sw.Vars.Global.Get(globalQuery)
	}

	tasks := unmarshalResultList(result)

	nreq := req.Clone()
	// single response with empty initial result
	nresp := new(api.Response)
	nresp.Result = &api.Result{
		Value: "",
	}
	const tpl = `
Accumulator:
	%s

CurrentValue:
	%s

Index:
	%v
	`
	for i, v := range tasks {
		nreq.RawInput = &api.UserInput{
			Message: fmt.Sprintf(tpl, nresp.Result.Value, v, i),
		}
		if err := h.flowSequence(nreq, nresp); err != nil {
			nresp.Result = &api.Result{
				Value: err.Error(),
			}
		}
	}

	resp.Result = nresp.Result
	return nil
}

// FlowTypeScript delegates control to a shell script using bash script syntax, enabling
// complex flow control scenarios driven by external scripting logic.
func (h *agentHandler) flowScript(req *api.Request, resp *api.Response) error {
	if h.agent.Flow == nil || h.agent.Flow.Script == "" {
		return fmt.Errorf("missing script content for script flow: %v", h.agent.Name)
	}

	ctx := req.Context()

	result := h.runScript(ctx, h.agent.Flow.Script)
	resp.Result = &api.Result{
		Value: result,
	}

	return nil
}

func (h *agentHandler) newExecHandler(ioe *sh.IOE) sh.ExecHandler {

	return func(ctx context.Context, args []string) (bool, error) {
		log.GetLogger(ctx).Debugf("parent: %s args: %+v\n", h.agent.Name, args)
		if args[0] == "ai" || strings.HasPrefix(args[0], "@") {
			fmt.Fprintf(ioe.Stdout, "ai: %+v\n", args)
			return true, nil
		}
		// allow bash built in
		if interp.IsBuiltin(args[0]) {
			return false, nil
		}
		// block other commands
		return true, nil
	}
}

func (h *agentHandler) runScript(ctx context.Context, script string) string {
	prStdout, pwStdout := io.Pipe()
	prStderr, pwStderr := io.Pipe()
	defer prStdout.Close()
	defer prStderr.Close()

	ioe := &sh.IOE{Stdin: nil, Stdout: pwStdout, Stderr: pwStderr}
	defer pwStdout.Close()
	defer pwStderr.Close()

	// global env
	//TODO
	vs := sh.NewVirtualSystem(h.sw.OS, h.sw.Workspace, ioe)
	vs.ExecHandler = h.newExecHandler(ioe)
	vs.RunScript(ctx, script)

	outputChan := make(chan string)
	errorChan := make(chan string)

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, prStdout)
		outputChan <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, prStderr)
		errorChan <- buf.String()
	}()

	result := <-outputChan
	if err := <-errorChan; err != "" {
		result = fmt.Sprintf("%s\nError: %s", result, err)
	}
	return result
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
