package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type Agent interface {
	Send(ctx context.Context, input string) (*ChatMessage, error)
}

type ChatMessage struct {
	Agent   string
	Content string
}

func MakeAgent(name string, cfg *llm.Config, role, content string) (Agent, error) {
	switch name {
	case "ask":
		return NewAskAgent(cfg, role, content)
	case "eval":
		return NewEvalAgent(cfg, role, content)
	case "seek":
		return NewGptrAgent(cfg, role, content)
	case "sql":
		return NewSqlAgent(cfg, role, content)
	case "gptr":
		return NewGptrAgent(cfg, role, content)
	case "oh":
		return NewOhAgent(cfg, role, content)
	case "git":
		return NewGitAgent(cfg, role, content)
	case "code":
		return NewAiderAgent(cfg, role, content)
	default:
		return nil, internal.NewUserInputError("not supported yet: " + name)
	}
}

func agentList() (map[string]string, error) {
	return resource.AgentDesc, nil
}

func HandleCommand(cfg *llm.Config, role, content string) error {
	log.Debugf("Handle: %s %v\n", cfg.Command, cfg.Args)

	command := cfg.Command

	if command != "" {
		// $ ai /command
		if strings.HasPrefix(command, "/") {
			return SlashCommand(cfg, role, content)
		}

		// $ ai @agent
		if strings.HasPrefix(command, "@") {
			return AgentCommand(cfg, role, content)
		}
	}

	// auto select the best agent to handle the user query if there is message content
	// $ ai message...
	return AgentHelp(cfg, role, content)
}

// resolveWorkspaceBase resolves the workspace base path.
// If the workspace is provided, validate and create if needed and return it.
// If the workspace is not provided, it tries to detect the workspace from the input using LLM.
// If the workspace is under the current directory (sub module), use the current directory as the workspace.
// If the workspace or its parent is a git repo (inside a git repo), use that as the workspace.
func resolveWorkspaceBase(ctx context.Context, cfg *llm.Config, workspace string, input string) (string, error) {
	if workspace != "" {
		return validatePath(workspace)
	}

	var err error
	if workspace, err = llm.DetectWorkspace(ctx, &llm.Model{
		Name:    cfg.L1Model,
		BaseUrl: cfg.L1BaseUrl,
		ApiKey:  cfg.L1ApiKey,
	}, input); err != nil {
		return "", err
	}

	log.Infof("Workspace detected: %s\n", workspace)

	if workspace, err = validatePath(workspace); err != nil {
		return "", err
	}

	// if the workspace contains the current working directory, use the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(workspace, cwd) {
		return cwd, nil
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
	for {
		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			return path, nil
		}
		path = filepath.Dir(path)
		if path == "/" {
			break
		}
	}

	return path, nil
}
