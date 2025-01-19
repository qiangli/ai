package agent

import (
	"context"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/util"
)

type ScriptAgent struct {
	config *llm.Config

	Role   string
	Prompt string
}

func NewScriptAgent(cfg *llm.Config, role, prompt string) (*ScriptAgent, error) {
	if role == "" {
		role = "system"
	}
	var err error
	info, err := util.CollectSystemInfo()
	if err != nil {
		return nil, err
	}
	if prompt == "" {
		prompt, err = resource.GetShellSystemRoleContent(info)
		if err != nil {
			return nil, err
		}
	}

	cfg.Tools = llm.GetSystemTools()

	chat := ScriptAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &chat, nil
}

func (r *ScriptAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	userContent, err := resource.GetShellUserRoleContent(
		in.Command, in.Input(),
	)
	if err != nil {
		return nil, err
	}

	resp, err := llm.Chat(ctx, &llm.Message{
		Role:    r.Role,
		Prompt:  r.Prompt,
		Model:   llm.Level2(r.config),
		Input:   userContent,
		DBCreds: r.config.Sql.DBConfig,
	})
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "SLASH",
		Content: resp.Content,
	}, nil
}
