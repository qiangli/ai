package swarm

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sort"
	"strings"

	// "github.com/qiangli/ai/internal/log"
	// utool "github.com/qiangli/ai/internal/tool"
	"github.com/qiangli/ai/swarm/api"
)

type FuncKit struct {
}

// func (r *FuncKit) FetchLocation(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	return utool.FetchLocation()
// }

func (r *FuncKit) ListAgents(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
	var list []string
	if vars.Config.AgentLister != nil {
		dict, err := vars.Config.AgentLister()
		if err != nil {
			return "", err
		}
		for k, v := range dict {
			list = append(list, fmt.Sprintf("%s: %s", k, v.Agents[0].Description))
		}
		sort.Strings(list)
	}
	return fmt.Sprintf("Available agents:\n%s\n", strings.Join(list, "\n")), nil
}

func (r *FuncKit) AgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return "", err
	}
	if vars.Config.AgentLister != nil {
		dict, err := vars.Config.AgentLister()
		if err != nil {
			return "", err
		}
		if v, ok := dict[agent]; ok {
			var desc []string
			for _, a := range v.Agents {
				desc = append(desc, a.Description)
			}
			return fmt.Sprintf("Agent: %s\nDescription: %s\n", v.Name, strings.Join(desc, "\n")), nil
		}
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

// TODO
func callAgentTransfer(_ context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}

// func (r *FuncKit) AskQuestion(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	question, err := GetStrProp("question", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return getUserTextInput(question)
// }

// func (r *FuncKit) TaskComplete(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	log.Infof("✌️ task completed %s", name)
// 	return "Task completed", nil
// }

func callFuncTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (string, error) {
	tool := &FuncKit{}
	callArgs := []any{ctx, vars, f.Name, args}
	v, err := CallKit(tool, f.Config.Kit, f.Name, callArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to call function tool %s %s: %w", f.Config.Kit, f.Name, err)
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", v), nil
}
