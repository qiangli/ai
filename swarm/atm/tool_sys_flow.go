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
func (r *SystemKit) Flow(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	flowType := api.FlowType(api.ToString(argm["flow_type"]))
	// default
	if flowType == "" {
		flowType = api.FlowTypeSequence
	}
	switch flowType {
	case api.FlowTypeSequence:
		return r.Sequence(ctx, vars, "", argm)
	case api.FlowTypeParallel:
		return r.Parallel(ctx, vars, "", argm)
	case api.FlowTypeChoice:
		return r.Choice(ctx, vars, "", argm)
	case api.FlowTypeMap:
		return r.Map(ctx, vars, "", argm)
	case api.FlowTypeChain:
		return r.Chain(ctx, vars, "", argm)
	default:
		return nil, fmt.Errorf("flow type not supported: %s", flowType)
	}
}

// FlowTypeSequence executes actions one after another, where each
// subsequent action uses the previous action's output as input.
func (r *SystemKit) Sequence(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	// var query = argm.Query()
	var actions = argm.Actions()

	result, err := r.sequence(ctx, vars, "", actions, argm)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *SystemKit) sequence(ctx context.Context, vars *api.Vars, _ string, actions []string, argm api.ArgMap) (*api.Result, error) {
	var result any
	for _, v := range actions {
		data, err := vars.RootAgent.Runner.Run(ctx, v, argm)
		if err != nil {
			return nil, err
		}
		result = data
	}
	return api.ToResult(result), nil
}

// FlowTypeParallel executes actions simultaneously, returning the combined results as a list.
// This allows for concurrent processing of independent actions.
func (r *SystemKit) Parallel(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
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
		return nil, err
	}
	return api.ToResult(data), nil
}

// FlowTypeChoice selects and executes a single action based on an evaluated expression.
// If no expression is provided, an action is chosen randomly. The expression must evaluate
// to a string (tool ID), false/true, or an integer that selects the action index, starting from zero.
func (r *SystemKit) Choice(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	var actions = argm.Actions()
	var which int = -1

	which = rand.Intn(len(actions))

	v := actions[which]
	data, err := vars.RootAgent.Runner.Run(ctx, v, argm)
	if err != nil {
		return nil, err
	}
	result := api.ToResult(data)
	return result, nil
}

// FlowTypeMap applies specified action(s) to each element in the input array, creating a new
// array populated with the results.
// similar to xargs utility
func (r *SystemKit) Map(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	var query = argm.Query()
	if query == "" {
		return nil, fmt.Errorf("missing query.")
	}
	var actions = argm.Actions()
	if len(actions) == 0 {
		return nil, fmt.Errorf("missing actions")
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
		return nil, err
	}
	return api.ToResult(data), nil
}

// // FlowTypeShell delegates control to a shell script using bash script syntax, enabling
// // complex flow control scenarios driven by external scripting logic.
// func (r *SystemKit) FlowShell(ctx context.Context, vars *api.Vars, argm api.ArgMap) (*api.Result, error)  {
// 	var script = argm.GetString("script")
// 	data, err := vars.RootAgent.Shell.Run(ctx, script, argm)
// 	if err != nil {
// 		return err
// 	}
// 	result := api.ToResult(data)
// 	return nil
// }

func (r *SystemKit) Chain(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	sa := argm.Actions()

	var actions []*api.ToolFunc
	for _, v := range sa {
		// only tool id (kit:name) is required
		// for running the chain.
		kit, name := api.Kitname(v).Decode()
		tool := &api.ToolFunc{
			Kit:  kit,
			Name: name,
		}
		actions = append(actions, tool)
	}
	return RunChainActions(ctx, vars, actions, argm)
}
