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
func evaluateCommand(ctx context.Context, agent *Agent, command string, args []string) (bool, error) {
	instruction, err := applyTemplate(shellSecuritySystemRole, agent.sw.Vars, nil)
	if err != nil {
		return false, err
	}

	agent.sw.Vars.Extra["Command"] = command
	agent.sw.Vars.Extra["Args"] = strings.Join(args, " ")
	query, err := applyTemplate(shellSecurityUserRole, agent.sw.Vars, nil)
	if err != nil {
		return false, err
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (*Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		out, err := runCommandTool(ctx, agent, name, args)
		if err != nil {
			return &Result{
				Value: fmt.Sprintf("%s: %v", out, err),
			}, nil
		}
		return &api.Result{
			Value: out,
		}, nil
	}

	req := &api.Request{
		ModelType: agent.Model.Type,
		Model:     agent.Model.Name,
		BaseUrl:   agent.Model.BaseUrl,
		ApiKey:    agent.Model.ApiKey,
		Messages: []*Message{
			{
				Role:    RoleSystem,
				Content: instruction,
			},
			{
				Role:    RoleUser,
				Content: query,
			},
		},
		Tools:    agent.Functions,
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
