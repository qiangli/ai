package swarm

import (
	"context"
	"fmt"
	"time"

	"github.com/briandowns/spinner"

	"github.com/qiangli/ai/internal/api"
)

func CallTool(ctx context.Context, vars *Vars, name string, args map[string]any) (*Result, error) {
	v, ok := vars.ToolMap[name]
	if !ok {
		return nil, fmt.Errorf("no such tool: %s", name)
	}

	// spinner
	sp := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	sp.Suffix = " calling " + name + "\n"

	switch v.Label {
	case ToolLabelAgent:
		nextAgent := v.Service
		if v.Func != "" {
			nextAgent = fmt.Sprintf("%s/%s", v.Service, v.Func)
		}
		return &api.Result{
			NextAgent: nextAgent,
			State:     api.StateTransfer,
		}, nil
	case ToolLabelMcp:
		sp.Start()
		defer sp.Stop()

		out, err := callMcpTool(ctx, vars, name, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolLabelSystem:
		out, err := callSystemTool(ctx, vars, name, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolLabelFunc:
		if fn, ok := vars.FuncRegistry[v.Func]; ok {
			return fn(ctx, vars, v.Func, args)
		}
	}

	return nil, fmt.Errorf("no such tool: %s", v.Name())
}
