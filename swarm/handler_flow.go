package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
)

func AgentFlowMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		var ah = &agentHandler{
			agent: agent,
			sw:    sw,
			next:  next,
		}
		// ah.initTemplate()
		// ah.initChain()
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			log.GetLogger(req.Context()).Debugf("🔗 (agent): %s flow: %+v\n", agent.Name, agent.Flow)

			return ah.Serve(req, resp)
		})
	}
}

type agentHandler struct {
	agent *api.Agent
	sw    *Swarm

	next api.Handler
}

// Serve calls the language model adapter with the messages list (after applying the system prompt).
// If the resulting response contains tool_calls, the tool runner will then call the tools.
// The tools kit executes the tools and adds the responses to the messages list.
// The adapter then calls the language model again.
// The process repeats until no more tool_calls are present in the response.
// The agent handler then returns the full list of messages.
func (h *agentHandler) Serve(req *api.Request, resp *api.Response) error {
	var ctx = req.Context()
	log.GetLogger(ctx).Debugf("Serve agent: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)

	// this needs to happen before everything else
	// h.sw.Vars.Global.Set(globalQuery, req.Query())

	if err := h.doAgentFlow(req, resp); err != nil {
		// h.sw.Vars.Global.Set(globalError, err.Error())
		return err
	}

	var result string
	if resp.Result != nil {
		result = resp.Result.Value
	}
	// if result is json, unpack for subsequnent agents/actions
	if len(result) > 0 {
		var resultMap = make(map[string]any)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			h.sw.Vars.Global.AddEnvs(resultMap)
		}
	}
	// h.sw.Vars.Global.Set(globalResult, result)

	log.GetLogger(ctx).Debugf("completed: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)
	return nil
}

// run agent first if there is instruction followed by the flow.
// otherwise, run the flow only
func (h *agentHandler) doAgentFlow(req *api.Request, resp *api.Response) error {
	instruction := h.agent.Instruction
	if instruction == "" && h.agent.Flow == nil {
		// no op?
		return api.NewBadRequestError("missing instruction and flow")
	}

	// run llm inference
	if instruction != "" {
		// if err := h.next.Serve(req, resp); err != nil {
		// 	return err
		// }
		if err := h.handleAgent(req, resp); err != nil {
			return err
		}
	}

	// flow control agent
	if h.agent.Flow != nil {
		if len(h.agent.Flow.Actions) == 0 && len(h.agent.Flow.Script) == 0 {
			return fmt.Errorf("missing actions or script in flow")
		}
		switch h.agent.Flow.Type {
		case api.FlowTypeSequence:
			if err := h.flowSequence(req, resp); err != nil {
				return err
			}
		case api.FlowTypeParallel:
			if err := h.flowParallel(req, resp); err != nil {
				return err
			}
		// case api.FlowTypeChoice:
		// 	if err := h.flowChoice(req, resp); err != nil {
		// 		return err
		// 	}
		case api.FlowTypeMap:
			if err := h.flowMap(req, resp); err != nil {
				return err
			}
		// case api.FlowTypeLoop:
		// 	if err := h.flowLoop(req, resp); err != nil {
		// 		return err
		// 	}
		// case api.FlowTypeReduce:
		// 	if err := h.flowReduce(req, resp); err != nil {
		// 		return err
		// 	}
		case api.FlowTypeShell:
			if err := h.flowShell(req, resp); err != nil {
				return err
			}
		default:
			return fmt.Errorf("not supported yet %v", h.agent.Flow)
		}
	}

	return nil
}

func (h *agentHandler) handleAgent(req *api.Request, resp *api.Response) error {
	maxHistory := req.Arguments.GetInt("max_history")
	maxSpan := req.Arguments.GetInt("max_span")

	logger := log.GetLogger(req.Context())
	logger.Debugf("🔗 (context): %s max_history: %v max_span: %v\n", h.agent.Name, maxHistory, maxSpan)

	var id = h.sw.ID
	// var env = sw.globalEnv()

	var history []*api.Message

	// 1. New System Message
	// system role prompt as first message
	prompt := h.agent.Prompt()
	if prompt != "" {
		v := &api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: prompt,
			Sender:  h.agent.Name,
		}
		history = append(history, v)
	}

	// 2. Context Messages
	// skip system role
	for i, msg := range h.agent.History() {
		if msg.Role != api.RoleSystem {
			logger.Debugf("adding [%v]: %s %s (%v)\n", i, msg.Role, abbreviate(msg.Content, 100), len(msg.Content))
			history = append(history, msg)
		}
	}

	// 3. New User Message
	// Additional user message
	var query = h.agent.Query()
	if query != "" {
		v := &api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			Role:    api.RoleUser,
			Content: query,
			Sender:  h.sw.User.Email,
		}
		history = append(history, v)
	}

	logger.Infof("• context messages: %v\n", len(history))
	if logger.IsTrace() {
		for i, v := range history {
			logger.Debugf("[%v] %+v\n", i, v)
		}
	}

	// request
	req.Name = h.agent.Name
	req.Tools = h.agent.Tools
	req.Runner = h.agent.Runner

	//
	initLen := len(history)
	req.Messages = history

	// call next
	if err := h.next.Serve(req, resp); err != nil {
		return err
	}

	if resp.Result == nil {
		resp.Result = &api.Result{}
	}
	var result = resp.Result

	// Response
	if result.State != api.StateTransfer {
		message := api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			ContentType: result.MimeType,
			Content:     result.Value,
			// TODO encode result.Result.Content
			Role:   nvl(result.Role, api.RoleAssistant),
			Sender: h.agent.Name,
		}
		// TODO add Value field to message?
		history = append(history, &message)
	}

	// h.sw.Vars.AddHistory(history)
	h.agent.SetHistory(history)
	//
	resp.Messages = history[initLen:]
	resp.Agent = h.agent
	resp.Result = result

	return nil
}

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
		h.agent.SetQuery(nresp.Result.Value)
		// h.sw.Vars.Global.Set(globalQuery, nresp.Result.Value)
	}

	// final result
	resp.Result = nresp.Result
	return nil
}

// // FlowTypeLoop executes actions repetitively in a loop. The loop can use a counter or
// // evaluate an expression for each iteration, allowing for repeated execution with varying
// // parameters or conditions.
// func (h *agentHandler) flowLoop(req *api.Request, resp *api.Response) error {
// 	env := h.sw.globalEnv()
// 	// h.mapAssign(req, env, req.Arguments, false)

// 	eval := func(exp string) (bool, error) {
// 		v, err := atm.ApplyTemplate(h.agent.Template, exp, env)
// 		if err != nil {
// 			return false, err
// 		}
// 		return strconv.ParseBool(v)
// 	}

// 	for {
// 		ok, err := eval(h.agent.Flow.Expression)
// 		if err != nil {
// 			return err
// 		}
// 		if !ok {
// 			return nil
// 		}
// 		if ok {
// 			// use the same request and respone
// 			if err := h.flowSequence(req, resp); err != nil {
// 				return err
// 			}
// 		}
// 	}
// }

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
			nreq := req.Clone()
			nreq.Agent = req.Agent.Clone()
			nresp := new(api.Response)
			//
			if err := h.doAction(ctx, nreq, nresp, v); err != nil {
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

// // FlowTypeChoice selects and executes a single action based on an evaluated expression.
// // If no expression is provided, an action is chosen randomly. The expression must evaluate
// // to a string (tool ID), false/true, or an integer that selects the action index, starting from zero.
// func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
// 	env := h.sw.globalEnv()
// 	// h.mapAssign(req, env, req.Arguments, false)

// 	var which int = -1
// 	// evaluate express or random
// 	if h.agent.Flow.Expression != "" {
// 		v, err := atm.ApplyTemplate(h.agent.Template, h.agent.Flow.Expression, env)
// 		if err != nil {
// 			return err
// 		}
// 		// match the action id
// 		id := api.Kitname(v).ID()
// 		for i, action := range h.agent.Flow.Actions {
// 			if id == action.ID {
// 				which = i
// 			}
// 		}
// 		//
// 		if b, err := strconv.ParseBool(v); err == nil {
// 			if b {
// 				which = 1
// 			} else {
// 				which = 0
// 			}
// 		}
// 		if which < 0 {
// 			if v, err := strconv.ParseInt(v, 0, 64); err != nil {
// 				return err
// 			} else {
// 				which = int(v)
// 			}
// 		}
// 	} else {
// 		// random
// 		which = rand.Intn(len(h.agent.Flow.Actions))
// 	}
// 	if which < 0 && which >= len(h.agent.Flow.Actions) {
// 		return fmt.Errorf("index out of bound; %v", which)
// 	}

// 	ctx := req.Context()

// 	v := h.agent.Flow.Actions[which]
// 	if err := h.doAction(ctx, req, resp, v); err != nil {
// 		return err
// 	}
// 	return nil
// }

// FlowTypeMap applies specified action(s) to each element in the input array, creating a new
// array populated with the results.
func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	// if the map flow is the first in the pipeline
	// use query
	// result, ok := h.sw.Vars.Global.Get(globalResult)
	result := h.agent.Result()
	if result == "" {
		// result, _ = h.sw.Vars.Global.Get(globalQuery)
		// result = req.Query
		result = h.agent.Query()
	}

	tasks := unmarshalResultList(result)

	var resps = make([]*api.Response, len(tasks))

	var wg sync.WaitGroup
	for i, v := range tasks {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()

			nreq := req.Clone()
			nreq.Agent = req.Agent.Clone()
			nreq.Agent.SetQuery(v)
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
	// runner := NewAgentScriptRunner(h.sw, h.agent)

	// make a copy of the args which already include args from the agent
	var args = make(map[string]any)
	if req.Arguments != nil {
		req.Arguments.Copy(args)
	}

	data, err := h.sw.Shell.Run(ctx, h.agent.Flow.Script, args)
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
