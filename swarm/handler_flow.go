package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	// "io"
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/interp"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
)

// flow actions
func (h *agentHandler) doAction(ctx context.Context, req *api.Request, resp *api.Response, tf *api.ToolFunc) error {
	var r = h.agent

	args, err := h.globalEnv(req)
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
	if h.agent == nil || h.agent.Flow == nil {
		return fmt.Errorf("flow sequence missing flow actions: %s", req.Name)
	}
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
	if h.agent == nil || h.agent.Flow == nil {
		return fmt.Errorf("flow loop missing agent or flow config")
	}

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
	if h.agent == nil || h.agent.Flow == nil {
		return fmt.Errorf("flow parallel missing agent or flow config")
	}

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
	if h.agent == nil || h.agent.Flow == nil {
		return fmt.Errorf("flow chocie missing agent or flow config")
	}

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
	if h.agent == nil || h.agent.Flow == nil || h.agent.Flow.Script == "" {
		return fmt.Errorf("script flow missing agent or script: %v", req.Name)
	}

	ctx := req.Context()
	var b bytes.Buffer

	ioe := &sh.IOE{Stdin: nil, Stdout: &b, Stderr: &b}

	nreq := req.Clone()
	nresp := new(api.Response)

	vs := sh.NewVirtualSystem(h.sw.OS, h.sw.Workspace, ioe)

	// global env
	// TODO batch set?
	env, err := h.globalEnv(req)
	if err != nil {
		return err
	}
	for k, v := range env {
		vs.System.Setenv(k, v)
	}

	vs.ExecHandler = h.newExecHandler(nreq, nresp)

	if err := vs.RunScript(ctx, h.agent.Flow.Script); err != nil {
		return err
	}

	resp.Result = &api.Result{
		Value: b.String(),
	}

	return nil
}

func (h *agentHandler) newExecHandler(req *api.Request, resp *api.Response) sh.ExecHandler {
	var memo = h.buildAgentToolMap()
	return func(ctx context.Context, args []string) (bool, error) {
		if h.agent == nil {
			return true, fmt.Errorf("missing agent: %v", req.Name)
		}
		log.GetLogger(ctx).Debugf("parent: %s args: %+v\n", h.agent.Name, args)
		isAi := func(s string) bool {
			return s == "ai" || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "/")
		}
		if isAi(strings.ToLower(args[0])) {
			log.GetLogger(ctx).Debugf("running ai agent/tool: %+v\n", args)
			at, err := conf.ParseAgentToolArgs(h.agent.Owner, args)
			if err != nil {
				return true, err
			}
			// agent tool
			nreq := req.Clone()
			nresp := new(api.Response)

			var kit string
			var name string
			if at.Agent != nil {
				nreq.Parent = h.agent
				nreq.Name = at.Agent.Name
				nreq.RawInput = at.Agent.RawInput
				nreq.Arguments = at.Agent.Arguments
				kit = "agent"
				name = nvl(at.Agent.Name, "anonymous")
			} else if at.Tool != nil {
				nreq.Parent = h.agent
				nreq.Name = at.Tool.Name
				nreq.RawInput = &api.UserInput{}
				nreq.Arguments = at.Tool.Arguments
				kit = at.Tool.Kit
				name = at.Tool.Name
			} else {
				// dicard
				return true, nil
			}
			id := api.KitName(kit + ":" + name).ID()
			v, ok := memo[id]
			if !ok {
				return true, fmt.Errorf("agent tool not declared for %s: %s", h.agent.Name, id)
			}
			return true, h.doAction(ctx, nreq, nresp, v)
		}
		// allow bash built in
		if interp.IsBuiltin(args[0]) {
			return false, nil
		}
		// block other commands
		return true, nil
	}
}

// func (h *agentHandler) runScript(ctx context.Context, req *api.Request, resp *api.Response, script string) error {
// 	// prStdout, pwStdout := io.Pipe()
// 	// prStderr, pwStderr := io.Pipe()
// 	// defer pwStdout.Close()
// 	// defer pwStderr.Close()

// 	var b bytes.Buffer
// 	// c.Stdout = &b
// 	// c.Stderr = &b

// 	ioe := &sh.IOE{Stdin: nil, Stdout: &b, Stderr: &b}

// 	// // ioe := &sh.IOE{Stdin: nil, Stdout: pwStdout, Stderr: pwStderr}

// 	// global env
// 	//TODO
// 	nreq := req.Clone()
// 	nresp := new(api.Response)

// 	vs := sh.NewVirtualSystem(h.sw.OS, h.sw.Workspace, ioe)
// 	vs.ExecHandler = h.newExecHandler(nreq, nresp)

// 	if err := vs.RunScript(ctx, script); err != nil {
// 		return err
// 	}

// 	// // outputChan := make(chan string)
// 	// // errorChan := make(chan string)

// 	// // go func() {
// 	// // 	var buf bytes.Buffer
// 	// // 	io.Copy(&buf, prStdout)
// 	// // 	prStdout.Close()
// 	// // 	outputChan <- buf.String()
// 	// // 	close(outputChan)
// 	// // }()

// 	// // go func() {
// 	// // 	var buf bytes.Buffer
// 	// // 	io.Copy(&buf, prStderr)
// 	// // 	prStderr.Close()
// 	// // 	errorChan <- buf.String()
// 	// // 	close(errorChan)
// 	// // }()

// 	// // var result, stderr string
// 	// // select {
// 	// // case out := <-outputChan:
// 	// // 	result = out
// 	// // case err := <-errorChan:
// 	// // 	stderr = err
// 	// // }

// 	// if stderr != "" {
// 	// 	result = fmt.Sprintf("%s\nError: %s", result, stderr)
// 	// }

// 	resp.Result = &api.Result{
// 		Value: b.String(),
// 	}
// 	return nil
// }

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
