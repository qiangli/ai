package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/util"
)

type ScriptAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewScriptAgent(cfg *llm.Config, role, content string) (*ScriptAgent, error) {
	if role == "" {
		role = "system"
	}
	info, err := util.CollectSystemInfo()
	if err != nil {
		return nil, err
	}
	if content == "" {
		systemMessage, err := resource.GetShellSystemRoleContent(info)
		if err != nil {
			return nil, err
		}
		content = systemMessage
	}

	cfg.Tools = llm.GetSystemTools()

	chat := ScriptAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &chat, nil
}

func (r *ScriptAgent) Send(ctx context.Context, command, input string) (*ChatMessage, error) {
	userContent, err := resource.GetShellUserRoleContent(
		command, input,
	)
	if err != nil {
		return nil, err
	}

	resp, err := llm.Chat(ctx, &llm.Message{
		Role:    r.Role,
		Prompt:  r.Message,
		Model:   llm.Level2(r.config),
		Input:   userContent,
		DBCreds: r.config.DBConfig,
	})
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "SLASH",
		Content: resp.Content,
	}, nil
}
