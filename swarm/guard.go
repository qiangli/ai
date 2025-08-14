package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/internal/bubble/confirm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/api/model"
	"github.com/qiangli/ai/swarm/llm"
)

//go:embed resource/shell_security_system.md
var shellSecuritySystemRole string

//go:embed resource/shell_security_user.md
var shellSecurityUserRole string

const permissionDenied = "Permission denied."

type CommandCheck struct {
	Command string `json:"command"`
	Safe    bool   `json:"safe"`
}

// evaluateCommand consults LLM to evaluate the safety of a command
func evaluateCommand(ctx context.Context, vars *api.Vars, command string, args []string) (bool, error) {
	if vars.Config.Unsafe {
		log.Infof("⚠️ unsafe mode - skipping security check\n")
		return true, nil
	}

	log.Infof("🔒 checking %s %+v\n", command, args)

	instruction, err := applyTemplate(shellSecuritySystemRole, vars, nil)
	if err != nil {
		return false, err
	}

	vars.Extra["Command"] = command
	vars.Extra["Args"] = strings.Join(args, " ")
	query, err := applyTemplate(shellSecurityUserRole, vars, nil)
	if err != nil {
		return false, err
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		out, err := CallTool(ctx, vars, name, args)
		return out, err
	}

	// TODO default model
	// m, ok := vars.Models[model.L1]
	// if !ok {
	// 	m = vars.Models[model.L2]
	// }
	m, err := vars.Config.ModelLoader(model.Any)
	if err != nil {
		return false, fmt.Errorf("failed to load model: %v", err)
	}

	req := &api.LLMRequest{
		Model: m,
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
		Tools:    vars.Config.SystemTools,
		RunTool:  runTool,
		MaxTurns: vars.Config.MaxTurns,
	}

	log.Debugf("evaluateCommand:\n%s %v\n", command, args)

	resp, err := llm.Send(ctx, req)
	if err != nil {
		return false, err
	}

	var check CommandCheck
	if err := json.Unmarshal([]byte(resp.Content), &check); err != nil {
		return false, fmt.Errorf("%s %s: %s, %s", command, strings.Join(args, " "), permissionDenied, resp.Content)
	}

	if check.Safe {
		log.Infof("✔ safe\n")
	} else {
		log.Errorf("\n\033[31m✗\033[0m unsafe\n")
		log.Infof("%s %v\n\n", command, strings.Join(args, " "))
		if answer, err := bubble.Confirm("Continue?"); err == nil && answer == confirm.Yes {
			check.Safe = true
		}
	}

	log.Debugf("evaluateCommand:\n%+v\n", check)

	return check.Safe, nil
}
