package atm

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"

	"github.com/qiangli/ai/swarm/api"
)

// run agent first if there is instruction followed by the flow.
// otherwise, run the flow only
func (r *SystemKit) Flow(ctx context.Context, vars *api.Vars, name string, args map[string]any) (any, error) {
	flowType := api.FlowType(api.ToString(args["flow_type"]))
	// default
	if flowType == "" {
		flowType = api.FlowTypeSequence
	}
	switch flowType {
	case api.FlowTypeSequence:
		if err := r.FlowSequence(ctx, vars, args); err != nil {
			return "", err
		}
	case api.FlowTypeParallel:
		if err := r.FlowParallel(ctx, vars, args); err != nil {
			return "", err
		}
	case api.FlowTypeChoice:
		if err := r.FlowChoice(ctx, vars, args); err != nil {
			return "", err
		}
	case api.FlowTypeMap:
		if err := r.FlowMap(ctx, vars, args); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("flow type not supported: %s", flowType)
	}

	return args["result"], nil
}

// FlowTypeSequence executes actions one after another, where each
// subsequent action uses the previous action's output as input.
func (r *SystemKit) FlowSequence(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	var query = argm.Query()
	var actions = argm.Actions()

	_, err := r.sequence(ctx, vars, query, actions, argm)
	return err
}

func (r *SystemKit) sequence(ctx context.Context, vars *api.Vars, query string, actions []string, argm api.ArgMap) (*api.Result, error) {
	// argm["query"] = query

	var result *api.Result
	for _, v := range actions {
		// subsequent action uses the previous action's response as input.
		// if result != nil {
		// 	argm["query"] = result.Value
		// }
		data, err := vars.RootAgent.Runner.Run(ctx, v, argm)
		if err != nil {
			return nil, err
		}
		// if err != nil {
		// 	argm["error"] = err.Error()
		// 	return nil, err
		// }
		result = api.ToResult(data)
		// argm["result"] = result.Value
	}
	return result, nil
}

// FlowTypeParallel executes actions simultaneously, returning the combined results as a list.
// This allows for concurrent processing of independent actions.
func (r *SystemKit) FlowParallel(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	var query = argm.Query()
	var actions = argm.Actions()

	var resps = make([]string, len(actions))

	var wg sync.WaitGroup
	for i, v := range actions {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()
			data, err := r.sequence(ctx, vars, query, actions, argm)
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
func (r *SystemKit) FlowChoice(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
	var actions = argm.Actions()
	var which int = -1

	which = rand.Intn(len(actions))

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
func (r *SystemKit) FlowMap(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
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

			data, err := r.sequence(ctx, vars, query, actions, argm)
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
// func (r *SystemKit) FlowShell(ctx context.Context, vars *api.Vars, argm api.ArgMap) error {
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
