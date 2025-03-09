package agent

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

const missingWorkspace = "Please specify a workspace base directory."

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	Detected      bool   `json:"detected"`
}

// resolveWorkspaceAdvice resolves the workspace base path.
// If the workspace is provided, validate and create if needed and return it.
// If the workspace is not provided, it tries to detect the workspace from the input using LLM.
// If the workspace or its parent is a git repo (inside a git repo), use that as the workspace.
// func resolveWorkspaceBase(ctx context.Context, cfg *internal.LLMConfig, workspace string, input string) (string, error) {
func resolveWorkspaceAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, next swarm.Advice) error {
	var workspace = vars.Workspace
	var err error

	// if the workspace is provided, validate and create if needed and return it
	if workspace != "" {
		workspace, err = validatePath(workspace)
		if err != nil {
			return fmt.Errorf("failed to validate workspace: %w", err)
		}
		vars.Workspace = workspace
		return nil
	}

	//
	tpl, ok := resource.Prompts["workspace_user_role"]
	if !ok {
		return fmt.Errorf("failed to get template: %s", "ws_base_user_role")
	}
	query, err := applyTemplate(tpl, map[string]string{
		"Input": req.RawInput.Intent(),
	})
	if err != nil {
		return err
	}

	msg := &swarm.Message{
		Role:    api.RoleUser,
		Content: query,
		Sender:  req.Agent,
	}
	req.Message = msg
	if err := next(vars, req, resp, next); err != nil {
		return err
	}
	result := resp.LastMessage()

	var wsCheck WorkspaceCheck
	if err := json.Unmarshal([]byte(result.Content), &wsCheck); err != nil {
		return fmt.Errorf("%s: %w", missingWorkspace, err)
	}
	if !wsCheck.Detected {
		return fmt.Errorf("%s", missingWorkspace)
	}

	log.Debugf("workspace check: %+v\n", wsCheck)

	workspace = wsCheck.WorkspaceBase

	log.Infof("Workspace to use: %s\n", workspace)

	workspace, err = resolveWorkspaceBase(workspace)
	if err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	}
	vars.Workspace = workspace

	return nil
}
