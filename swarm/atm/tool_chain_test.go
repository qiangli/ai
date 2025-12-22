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

func (r mockActionRunner) Run(_ context.Context, tid string, args map[string]any) (any, error) {
	v := fmt.Sprintf("test result from: %s %+v", tid, args)
	r.logger(">>>%s: %v", tid, v)
	return &api.Result{
		Value: v,
	}, nil
}

func TestServeChainActions(t *testing.T) {

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
		"kit0:analytics",
		"kit1:timeout",
		"kit2:backoff",
		"kit3:ratelimit",
		"kit4:run_query",
	}

	result, err := RunChainActions(ctx, vars, actions, args)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", result)
}
