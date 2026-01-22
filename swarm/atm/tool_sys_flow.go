package atm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/util"
)

// FlowType Sequence executes actions one after another, where each
// subsequent action uses the previous action's output as input.
func (r *SystemKit) Sequence(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	var actions = argm.Actions()

	result, err := r.InternalSequence(ctx, vars, "", actions, argm)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// TODO merge with ai kit
func (r *SystemKit) InternalSequence(ctx context.Context, vars *api.Vars, _ string, actions []string, argm api.ArgMap) (*api.Result, error) {
	var result any
	for _, v := range actions {
		data, err := vars.RootAgent.Runner.Run(ctx, v, argm)
		if err != nil {
			argm["error"] = err
			return nil, err
		}
		argm["result"] = data
		result = data
	}
	return api.ToResult(result), nil
}

// FlowType Parallel executes actions simultaneously, returning the combined results as a list.
// This allows for concurrent processing of independent actions.
// all actions use the same input
func (r *SystemKit) Parallel(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	var actions = argm.Actions()

	var resps = make([]string, len(actions))
	// needed to prevent data race issues
	var nargs = make([]map[string]any, len(actions))

	// TODO lock machinism for actions to update args thread-safe?
	var wg sync.WaitGroup
	for i, v := range actions {
		wg.Add(1)
		go func(i int, v string) {
			defer wg.Done()
			nargs[i] = make(map[string]any)
			maps.Copy(nargs[i], argm)
			data, err := vars.RootAgent.Runner.Run(ctx, v, nargs[i])
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
	// copy back changes as if actions were run sequentially
	for _, v := range nargs {
		maps.Copy(argm, v)
	}
	return api.ToResult(string(data)), nil
}

// FlowType Choice selects and executes a single action based on an evaluated expression.
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

func (r *SystemKit) Chain(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	obj, ok := argm["chain"]
	if !ok {
		return nil, fmt.Errorf("chain actions is required")
	}
	actions := api.ToStringArray(obj)
	if len(actions) == 0 {
		return nil, fmt.Errorf("no actions specified")
	}
	out, err := StartChainActions(ctx, vars, actions, argm)
	if err != nil {
		return nil, err
	}
	return api.ToResult(out), nil
}

// FlowType Loop executes actions repetitively in a loop. The loop runs indefinitely or can use a counter.
func (r *SystemKit) Loop(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	var actions = argm.Actions()
	var result *api.Result
	var err error
	var max = argm.GetInt("max_iteration")
	if max <= 0 {
		max = math.MaxInt
	}
	duration := argm.GetString("sleep")
	sec, _ := util.ParseDuration(duration)
	if sec <= 0 {
		sec = 3 * time.Second
	}
	msg := argm.GetString("report")

	for i := 1; i < max; i++ {
		if msg != "" {
			log.GetLogger(ctx).Infof("%s\n", msg)
		}
		result, err = r.InternalSequence(ctx, vars, "", actions, argm)
		if err != nil {
			return nil, err
		}
		time.Sleep(sec)
	}
	return result, err
}

// FlowType Fallback executes actions in sequence. Return the result of the first successfully executed action, or produce an error from the final action if all actions fail.
func (r *SystemKit) Fallback(ctx context.Context, vars *api.Vars, _ string, argm api.ArgMap) (*api.Result, error) {
	var actions = argm.Actions()
	var respErr error
	for _, v := range actions {
		result, err := vars.RootAgent.Runner.Run(ctx, v, argm)
		if err == nil {
			return api.ToResult(result), nil
		}
		respErr = err
	}
	return nil, respErr
}
