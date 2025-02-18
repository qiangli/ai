package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

// resolveWorkspaceAdvice resolves the workspace base path.
// If the workspace is provided, validate and create if needed and return it.
// If the workspace is not provided, it tries to detect the workspace from the input using LLM.
// If the workspace is under the current directory (sub module), use the current directory as the workspace.
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
		Role:    swarm.RoleUser,
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

func resolveWorkspaceBase(workspace string) (string, error) {
	workspace, err := validatePath(workspace)
	if err != nil {
		return "", err
	}

	// if the workspace contains the current working directory, use the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(workspace, cwd) {
		workspace = cwd
	}

	// check if the workspace path or any of its parent contains a git repository
	// if so, use the git repository as the workspace
	if workspace, err = detectGitRepo(workspace); err != nil {
		return "", err
	}

	return workspace, nil
}

// ValidatePath returns the absolute path of the given path.
// If the path is empty, it returns an error. If the path is not an absolute path,
// it converts it to an absolute path.
// If the path exists, it returns its absolute path.
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		path = absPath
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return "", fmt.Errorf("failed to create directory: %w", err)
			}
			return path, nil
		}
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	return path, nil
}

func detectGitRepo(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	original := path
	for {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			return path, nil
		}
		np := filepath.Dir(path)
		if np == path || np == "/" {
			break
		}
		path = np
	}
	return original, nil
}

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	Detected      bool   `json:"detected"`
}

const missingWorkspace = "Please specify a workspace base directory."
const permissionDenied = "Permission denied."
