package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	Detected      bool   `json:"detected"`
}

const missingWorkspace = "Please specify a workspace base directory."
const permissionDenied = "Permission denied."

// DetectWorkspace decides the workspace with the help from LLM
func DetectWorkspace(ctx context.Context, model *internal.Model, input string) (string, error) {
	userContent, err := resource.GetWSBaseUserRoleContent(
		input,
	)
	if err != nil {
		return "", err
	}

	req := &internal.Message{
		Role:   "system",
		Prompt: resource.GetWSBaseSystemRoleContent(),
		Model:  model,
		Input:  userContent,
	}
	if model.Tools == nil {
		model.Tools = GetRestrictedSystemTools()
	}

	resp, err := Chat(ctx, req)
	if err != nil {
		return "", err
	}

	var wsCheck WorkspaceCheck
	if err := json.Unmarshal([]byte(resp.Content), &wsCheck); err != nil {
		return "", fmt.Errorf("%s: %w", missingWorkspace, err)
	}
	if !wsCheck.Detected {
		return "", fmt.Errorf("%s", missingWorkspace)
	}

	log.Debugf("WorkspaceCheck: %+v\n", wsCheck)

	return wsCheck.WorkspaceBase, nil
}

type CommandCheck struct {
	Command string `json:"command"`
	Safe    bool   `json:"safe"`
}

// EvaluateCommand consults LLM to evaluate the safety of a command
func EvaluateCommand(ctx context.Context, model *internal.Model, command string, args []string) (bool, error) {
	const tpl = "Here is the command and arguments: %s %s"
	req := &internal.Message{
		Role:   "system",
		Prompt: resource.GetShellSecurityRoleContent(),
		Model:  model,
		Input:  fmt.Sprintf(tpl, command, strings.Join(args, " ")),
	}

	if model.Tools == nil {
		model.Tools = GetRestrictedSystemTools()
	}

	resp, err := Chat(ctx, req)
	if err != nil {
		return false, err
	}

	var check CommandCheck
	if err := json.Unmarshal([]byte(resp.Content), &check); err != nil {
		return false, fmt.Errorf("%s %s: %s, %s", command, strings.Join(args, " "), permissionDenied, resp.Content)
	}

	log.Debugf("CommandCheck: %+v\n", check)

	return check.Safe, nil
}

// GenerateConfig extracts config settings from the input with the help from LLM.
func GenerateConfig(ctx context.Context, model *internal.Model, input string) (*resource.ConfigSchema, error) {
	prompt, err := resource.GetCliConfigSystem()
	if err != nil {
		return nil, err
	}
	userContent, err := resource.GetCliConfigUser(input)
	if err != nil {
		return nil, err
	}

	req := &internal.Message{
		Role:   "system",
		Prompt: prompt,
		Model:  model,
		Input:  userContent,
	}

	if model.Tools == nil {
		model.Tools = GetRestrictedSystemTools()
	}

	resp, err := Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	var cfg resource.ConfigSchema

	if err := json.Unmarshal([]byte(resp.Content), &cfg); err != nil {
		return nil, fmt.Errorf("failed to generate config: %w", err)
	}

	log.Debugf("config: %+v\n", cfg)

	return &cfg, nil
}
