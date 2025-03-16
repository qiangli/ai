package swarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
)

func EvalTool(ctx context.Context, vars *Vars, name string, args map[string]any) (*Result, error) {
	log.Infof("🔒 checking %s %+v\n", name, args)

	result, err := dispatchTool(ctx, vars, name, args)

	if err != nil {
		log.Errorf("❌ unsafe %s\n", err)
	} else {
		log.Infof("✅ safe %s\n", head(result.Value, 80))
	}

	return result, err
}

func CallTool(ctx context.Context, vars *Vars, name string, args map[string]any) (*Result, error) {
	log.Infof("✨ %s %+v\n", name, args)

	result, err := dispatchTool(ctx, vars, name, args)

	if err != nil {
		// log.Infof("❌ %s\n", err)
		log.Errorf("\033[31m✗\033[0m %s\n", err)
	} else {
		log.Infof("✔ %s\n", head(result.Value, 80))
	}

	return result, err
}

// head trims the string to the maxLen and replaces newlines with /.
func head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

func dispatchTool(ctx context.Context, vars *Vars, name string, args map[string]any) (*Result, error) {

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
