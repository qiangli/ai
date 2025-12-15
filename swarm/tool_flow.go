package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	// "strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/log"
)

type FlowKit struct {
}

// run agent first if there is instruction followed by the flow.
// otherwise, run the flow only
func (h *FlowKit) Flow(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	flowType := argm.GetString("flow_type")
	switch api.FlowType(flowType) {
	case api.FlowTypeSequence:
		if err := h.FlowSequence(ctx, vars, argm); err != nil {
			return err
		}
	case api.FlowTypeParallel:
		if err := h.FlowParallel(ctx, vars, argm); err != nil {
			return err
		}
	case api.FlowTypeChoice:
		if err := h.FlowChoice(ctx, vars, argm); err != nil {
			return err
		}
	case api.FlowTypeMap:
		if err := h.FlowMap(ctx, vars, argm); err != nil {
			return err
		}
	// case api.FlowTypeShell:
	// 	if err := h.FlowShell(ctx, vars, argm); err != nil {
	// 		return err
	// 	}
	default:
		return fmt.Errorf("not supported yet %s", flowType)
	}

	return nil
}

func (h *FlowKit) CallLlm(ctx context.Context, vars *api.Vars, agent *api.Agent, argm map[string]any) error {
	am := api.ArgMap(argm)
	maxHistory := am.GetInt("max_history")
	maxSpan := am.GetInt("max_span")

	logger := log.GetLogger(ctx)
	logger.Debugf("🔗 (context): %s max_history: %v max_span: %v\n", agent.Name, maxHistory, maxSpan)

	var id string
	var history []*api.Message

	// 1. New System Message
	// system role prompt as first message
	// prompt := h.agent.Prompt()
	var prompt = am.GetString("prompt")
	if prompt != "" {
		v := &api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			Role:    api.RoleSystem,
			Content: prompt,
			Sender:  agent.Name,
		}
		history = append(history, v)
	}

	// 2. Context Messages
	// skip system role
	var messages []*api.Message
	for i, msg := range messages {
		if msg.Role != api.RoleSystem {
			logger.Debugf("adding [%v]: %s %s (%v)\n", i, msg.Role, abbreviate(msg.Content, 100), len(msg.Content))
			history = append(history, msg)
		}
	}

	// 3. New User Message
	// Additional user message
	// var query = h.agent.Query()
	var query = am.GetString("query")
	if query != "" {
		v := &api.Message{
			ID:      uuid.NewString(),
			Session: id,
			Created: time.Now(),
			//
			Role:    api.RoleUser,
			Content: query,
			Sender:  vars.RTE.User.Email,
		}
		history = append(history, v)
	}

	logger.Infof("• context messages: %v\n", len(history))
	if logger.IsTrace() {
		for i, v := range history {
			logger.Debugf("[%v] %+v\n", i, v)
		}
	}

	am["history"] = history

	return nil
}

// FlowTypeSequence executes actions one after another, where each
// subsequent action uses the previous action's response as input.
func (h *FlowKit) FlowSequence(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	var query = argm.Query()
	var actions = argm.Actions()

	_, err := h.sequence(ctx, vars, query, actions, argm)
	return err
}

func (h *FlowKit) sequence(ctx context.Context, vars *api.Vars, query string, actions []string, argm api.ArgMap) (*api.Result, error) {
	if len(query) == 0 {
		return nil, fmt.Errorf("missing query")
	}
	//
	argm["query"] = query

	var result *api.Result
	for _, v := range actions {
		// subsequent action uses the previous action's response as input.
		if result != nil {
			argm["query"] = result.Value
		}
		data, err := vars.RootAgent.Runner.Run(ctx, v, argm)
		if err != nil {
			argm["error"] = err.Error()
			return nil, err
		}
		result = api.ToResult(data)
		argm["result"] = result.Value
	}
	return result, nil
}

// FlowTypeParallel executes actions simultaneously, returning the combined results as a list.
// This allows for concurrent processing of independent actions.
func (h *FlowKit) FlowParallel(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	var query = argm.Query()
	var actions = argm.Actions()

	var resps = make([]string, len(actions))

	var wg sync.WaitGroup
	for i, v := range actions {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()
			data, err := h.sequence(ctx, vars, query, actions, argm)
			if err != nil {
				resps[i] = err.Error()
			} else {
				result := api.ToResult(data)
				resps[i] = result.Value
			}
		}(i, v)
	}
	wg.Wait()

	data, err := json.Marshal(resps)
	if err != nil {
		return err
	}
	argm["result"] = string(data)
	return nil
}

// FlowTypeChoice selects and executes a single action based on an evaluated expression.
// If no expression is provided, an action is chosen randomly. The expression must evaluate
// to a string (tool ID), false/true, or an integer that selects the action index, starting from zero.
func (h *FlowKit) FlowChoice(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	// var query = argm.Query()
	var actions = argm.Actions()
	// var expression = argm.GetString("expression")
	var which int = -1
	// evaluate express or random
	// if expression != "" {
	// 	v, err := atm.CheckApplyTemplate(vars.RootAgent.Template, expression, argm)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	// match the action id
	// 	// switch case
	// 	id := api.Kitname(v).ID()
	// 	for i, action := range actions {
	// 		if id == action {
	// 			which = i
	// 		}
	// 	}
	// 	// if/else
	// 	if b, err := strconv.ParseBool(v); err == nil {
	// 		if b {
	// 			which = 1
	// 		} else {
	// 			which = 0
	// 		}
	// 	}
	// 	//  case
	// 	if which < 0 {
	// 		if v, err := strconv.ParseInt(v, 0, 64); err != nil {
	// 			return err
	// 		} else {
	// 			which = int(v)
	// 		}
	// 	}
	// } else {
	// 	// random
	// 	which = rand.Intn(len(actions))
	// }

	which = rand.Intn(len(actions))

	// which = which % len(actions)
	// if which < 0 && which >= len(actions) {
	// 	return fmt.Errorf("index out of bound; %v", which)
	// }

	v := actions[which]
	var result *api.Result
	data, err := vars.RootAgent.Runner.Run(ctx, v, argm)
	if err != nil {
		argm["error"] = err.Error()
		return err
	}
	result = api.ToResult(data)
	argm["result"] = result.Value
	return nil
}

// FlowTypeMap applies specified action(s) to each element in the input array, creating a new
// array populated with the results.
// similar to xargs utility
func (h *FlowKit) FlowMap(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	var query = argm.Query()
	if query == "" {
		return fmt.Errorf("missing query.")
	}
	var actions = argm.Actions()
	if len(actions) == 0 {
		return fmt.Errorf("missing actions")
	}

	var tasks []string
	err := json.Unmarshal([]byte(query), &tasks)
	if err != nil {
		// make is single task?
		// return err
		tasks = []string{query}
	}
	var resps = make([]string, len(tasks))

	var wg sync.WaitGroup
	for i, v := range tasks {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()

			var req = make(map[string]any)
			req["query"] = query

			data, err := h.sequence(ctx, vars, query, actions, argm)
			if err != nil {
				resps[i] = err.Error()
			} else {
				result := api.ToResult(data)
				resps[i] = result.Value
			}
		}(i, v)
	}
	wg.Wait()

	data, err := json.Marshal(resps)
	if err != nil {
		return err
	}
	argm["result"] = string(data)
	return nil
}

// // FlowTypeShell delegates control to a shell script using bash script syntax, enabling
// // complex flow control scenarios driven by external scripting logic.
// func (h *FlowKit) FlowShell(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
// 	var script = argm.GetString("script")
// 	data, err := vars.RootAgent.Shell.Run(ctx, script, argm)
// 	if err != nil {
// 		argm["error"] = err.Error()
// 		return err
// 	}
// 	result := api.ToResult(data)
// 	argm["result"] = result.Value
// 	return nil
// }
