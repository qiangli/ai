package agent

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm/agent/resource"
)

const missingWorkspace = "Please specify a workspace base directory."

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	Detected      bool   `json:"detected"`
}

// resolveWorkspaceAdvice resolves the workspace base path.
// Detect the workspace from the input using LLM.
// If the workspace or its parent is a git repo (inside a git repo), use that as the workspace.
// func resolveWorkspaceBase(ctx context.Context, cfg *internal.LLMConfig, workspace string, input string) (string, error) {
func resolveWorkspaceAdvice(vars *api.Vars, req *api.Request, resp *api.Response, next api.Advice) error {
	// var workspace = vars.Workspace
	// var err error

	// // if the workspace is provided, validate and create if needed and return it
	// if workspace != "" {
	// 	workspace, err = util.ValidatePath(workspace)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to validate workspace: %w", err)
	// 	}
	// 	if err := os.MkdirAll(workspace, os.ModePerm); err != nil {
	// 		return fmt.Errorf("failed to create directory: %w", err)
	// 	}
	// 	vars.Workspace = workspace
	// 	return nil
	// }

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

	msg := &api.Message{
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

	workspace := wsCheck.WorkspaceBase

	log.Infof("Workspace to use: %s\n", workspace)

	// // check if the workspace path or any of its parent contains a git repository
	// // if so, use the git repository as the workspace
	// workspace, err = util.DetectGitRepo(workspace)
	// if err != nil {
	// 	return err
	// }

	// workspace, err = util.ResolveWorkspace(workspace)
	// if err != nil {
	// 	return fmt.Errorf("failed to resolve workspace: %w", err)
	// }
	// vars.Workspace = workspace

	vars.Extra["workspace_base"] = workspace

	return nil
}
