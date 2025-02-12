package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

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
	systemRole, err := applyTemplate(shellSecuritySystemRole, agent.Vars, nil)
	if err != nil {
		return false, err
	}
	userRole, err := applyTemplate(shellSecurityUserRole, agent.Vars, nil)
	agent.Vars.Extra["Command"] = command
	agent.Vars.Extra["Args"] = strings.Join(args, " ")
	if err != nil {
		return false, err
	}
	req := &llm.Request{
		Model:   agent.Model.Name,
		BaseUrl: agent.Model.BaseUrl,
		ApiKey:  agent.Model.ApiKey,
		Messages: []*Message{
			{
				Role:    "system",
				Content: systemRole,
			},
			{
				Role:    "user",
				Content: userRole,
			},
		},
	}

	log.Debugf("evaluateCommand:\n%s %v\n", command, args)

	resp, err := llm.Call(ctx, req)
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
