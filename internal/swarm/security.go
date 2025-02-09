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

//go:embed resource/shell_security_role.md
var shellSecurityRole string

const permissionDenied = "Permission denied."

type CommandCheck struct {
	Command string `json:"command"`
	Safe    bool   `json:"safe"`
}

// EvaluateCommand consults LLM to evaluate the safety of a command
func EvaluateCommand(ctx context.Context, agent *Agent, command string, args []string) (bool, error) {
	content := fmt.Sprintf("Here is the command and arguments: %s %s", command, strings.Join(args, " "))
	req := &llm.Request{
		Model:   agent.Model.Name,
		BaseUrl: agent.Model.BaseUrl,
		ApiKey:  agent.Model.ApiKey,
		Messages: []*Message{
			{
				Role:    "system",
				Content: shellSecurityRole,
			},
			{
				Role:    "user",
				Content: content,
			},
		},
	}

	log.Debugf("EvaluateCommand:\n%s\n", content)

	resp, err := llm.Call(ctx, req)
	if err != nil {
		return false, err
	}

	var check CommandCheck
	if err := json.Unmarshal([]byte(resp.Content), &check); err != nil {
		return false, fmt.Errorf("%s %s: %s, %s", command, strings.Join(args, " "), permissionDenied, resp.Content)
	}

	log.Debugf("EvaluateCommand:\n%+v\n", check)

	return check.Safe, nil
}
