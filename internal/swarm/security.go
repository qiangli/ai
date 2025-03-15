package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
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
func evaluateCommand(ctx context.Context, vars *Vars, command string, args []string) (bool, error) {
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

	runTool := func(ctx context.Context, name string, args map[string]any) (*Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		out, err := CallTool(ctx, vars, name, args)
		return out, err
	}

	// TODO default model
	model, ok := vars.Models[api.L1]
	if !ok {
		model = vars.Models[api.L2]
	}
	if model == nil {
		return false, fmt.Errorf("no model found L1/L2")
	}

	req := &api.Request{
		ModelType: model.Type,
		Model:     model.Name,
		BaseUrl:   model.BaseUrl,
		ApiKey:    model.ApiKey,
		Messages: []*Message{
			{
				Role:    api.RoleSystem,
				Content: instruction,
			},
			{
				Role:    api.RoleUser,
				Content: query,
			},
		},
		Tools:    systemTools,
		RunTool:  runTool,
		MaxTurns: 32,
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

	log.Debugf("evaluateCommand:\n%+v\n", check)

	return check.Safe, nil
}
