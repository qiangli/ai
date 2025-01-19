package agent

import (
	"context"
	"path/filepath"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/util"
)

type ScriptAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewScriptAgent(cfg *internal.AppConfig) (*ScriptAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

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

	cfg.LLM.Tools = llm.GetSystemTools()

	chat := ScriptAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &chat, nil
}

func (r *ScriptAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	cmd := in.Subcommand
	if cmd != "" {
		cmd = filepath.Base(cmd)
	}

	userContent, err := resource.GetShellUserRoleContent(cmd, in.Input())
	if err != nil {
		return nil, err
	}

	msg := &internal.Message{
		Role:   r.Role,
		Prompt: r.Prompt,
		Model:  internal.Level2(r.config.LLM),
		Input:  userContent,
		// DBCreds: nil,
	}

	resp, err := llm.Chat(ctx, msg)

	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   in.Agent,
		Content: resp.Content,
	}, nil
}
