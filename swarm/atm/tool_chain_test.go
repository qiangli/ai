package atm

import (
	"context"
	"fmt"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

type mockActionRunner struct {
	logger func(string, ...any)
}

func (r mockActionRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	r.logger("\n******** running %s ********\n", tid)

	kit, name := api.Kitname(tid).Decode()
	switch kit {
	case "kit":
		action, ok := args["action"]
		if !ok {
			return nil, fmt.Errorf("no action found")
		}
		if v, ok := action.(string); ok {
			var result = &api.Result{}
			out, err := r.Run(ctx, v, args)
			result.Value = fmt.Sprintf("\n>>> [%v]\nout: %v\nerr: %v\n", action, out, err)
			return result, err
		}
		return nil, fmt.Errorf("action type not supported: %v", action)
	case "alias":
		break
	default:
		return nil, fmt.Errorf("not supported action: %s", tid)
	}

	// handle alias
	action, ok := args[name]
	if !ok {
		return nil, fmt.Errorf("alias %q not provided in arguments for: %s", name, tid)
	}
	if v, ok := action.(api.ActionRunner); ok {
		return v.Run(ctx, tid, args)
	}
	// other types of actions
	return nil, fmt.Errorf("failed to run alias: %s. not an action runner/can not be resolved", tid)
}

func TestStartChainActions(t *testing.T) {
	ctx := context.Background()
	args := api.NewArgMap()

	rootAgent := &api.Agent{
		Runner: mockActionRunner{
			logger: t.Logf,
		},
	}
	vars := &api.Vars{RootAgent: rootAgent}

	// BI: logging, analytics, and debugging.
	// Infra: retries, fallbacks, timeout, early termination.
	// Policy: rate limits, guardrails, pii detection.
	// Query: prompts, tool selection, and output formatting.
	actions := []string{
		"kit:analytics",
		"kit:timeout",
		"kit:backoff",
		"kit:ratelimit",
		"kit:run_query",
	}

	result, err := StartChainActions(ctx, vars, actions, args)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("\n\n Result:\n%v", result)
}
