package agent

import (
	"context"
	"encoding/json"
	"fmt"
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

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	Detected      bool   `json:"detected"`
}

const missingWorkspace = "Please specify a workspace base directory."

// Decide the workspace with the help from LLM
func checkWorkspace(ctx context.Context, cfg *llm.Config, input string, level llm.Level) (string, error) {
	ws := cfg.Workspace
	if ws == "" {

		userContent, err := resource.GetWSBaseUserRoleContent(
			input,
		)
		if err != nil {
			return "", err
		}

		role := "system"
		prompt := resource.GetWSBaseSystemRoleContent()

		req := &llm.Message{
			Role:    role,
			Prompt:  prompt,
			Model:   llm.CreateModel(cfg, level),
			Input:   userContent,
			DBCreds: cfg.DBConfig,
		}

		// role, prompt, userContent

		resp, err := llm.Chat(ctx, req)
		if err != nil {
			return "", err
		}
		// unmarshal the response
		// TODO: retry?
		var wsCheck WorkspaceCheck
		if err := json.Unmarshal([]byte(resp.Content), &wsCheck); err != nil {
			return "", fmt.Errorf("failed to unmarshal response: %w", err)
		}
		if !wsCheck.Detected {
			return "", fmt.Errorf("%s", missingWorkspace)
		}

		log.Debugf("Workspace check: %+v\n", wsCheck)

		ws = wsCheck.WorkspaceBase
	}

	// double check the workspace it is valid
	workspace, err := ValidatePath(ws)
	if err != nil {
		return "", err
	}

	log.Infof("Workspace: %s %s\n", ws, workspace)
	return workspace, nil
}
