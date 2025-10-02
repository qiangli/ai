package atm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/internal/bubble/confirm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/resource"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
)

// //go:embed resource/shell_security_system.md
// var shellSecuritySystemRole string

// //go:embed resource/shell_security_user.md
// var shellSecurityUserRole string

const permissionDenied = "Permission denied."

type CommandCheck struct {
	Command string `json:"command"`
	Safe    bool   `json:"safe"`
}

// evaluateCommand consults LLM to evaluate the safety of a command
func EvaluateCommand(ctx context.Context, vars *api.Vars, agent *api.Agent, command string, args []string) (bool, error) {
	if vars.Config.Unsafe {
		log.GetLogger(ctx).Infof("⚠️ unsafe mode - skipping security check\n")
		return true, nil
	}

	log.GetLogger(ctx).Infof("🔒 checking %s %+v\n", command, args)

	instruction, err := applyTemplate(resource.ShellSecuritySystemRole, vars, nil)
	if err != nil {
		return false, err
	}

	vars.Extra["Command"] = command
	vars.Extra["Args"] = strings.Join(args, " ")
	query, err := applyTemplate(resource.ShellSecurityUserRole, vars, nil)
	if err != nil {
		return false, err
	}

	// toolsMap := make(map[string]*api.ToolFunc)
	// for _, v := range vars.Config.SystemTools {
	// 	toolsMap[v.ID()] = v
	// }

	// runTool := func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
	// 	log.GetLogger(ctx).Debugf("run tool: %s %+v\n", name, args)
	// 	v, ok := toolsMap[name]
	// 	if !ok {
	// 		return nil, fmt.Errorf("not found: %s", name)
	// 	}
	// 	out, err := callTool(ctx, vars, agent, v, name, args)
	// 	return out, err
	// }

	req := &llm.Request{
		Model: agent.Model,
		Messages: []*api.Message{
			{
				Role:    api.RoleSystem,
				Content: instruction,
			},
			{
				Role:    api.RoleUser,
				Content: query,
			},
		},
		// Tools:    vars.Config.SystemTools,
		// RunTool:  runTool,
		MaxTurns: vars.Config.MaxTurns,
		Vars:     vars,
	}

	log.GetLogger(ctx).Debugf("evaluateCommand:\n%s %v\n", command, args)

	resp, err := adapter.Chat(ctx, req)
	if err != nil {
		return false, err
	}

	var check CommandCheck
	if err := json.Unmarshal([]byte(resp.Content), &check); err != nil {
		return false, fmt.Errorf("%s %s: %s, %s", command, strings.Join(args, " "), permissionDenied, resp.Content)
	}

	if check.Safe {
		log.GetLogger(ctx).Infof("✔ safe\n")
	} else {
		log.GetLogger(ctx).Errorf("❌ unsafe\n")
		log.GetLogger(ctx).Infof("%s %v\n", command, strings.Join(args, " "))
		if answer, err := bubble.Confirm("Continue?"); err == nil && answer == confirm.Yes {
			check.Safe = true
		}
	}

	log.GetLogger(ctx).Debugf("evaluateCommand:\n%+v\n", check)

	return check.Safe, nil
}
