package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type GitAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewGitAgent(cfg *internal.AppConfig) (*GitAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}
	agent := GitAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *GitAgent) getSystemPrompt(in *UserInput) (string, error) {
	if r.Prompt != "" {
		return r.Prompt, nil
	}
	switch baseCommand(in.Subcommand) {
	case "short":
		return resource.GetGitMessageSystem(true), nil
	case "conventional":
		return resource.GetGitMessageSystem(false), nil
	}
	return "", fmt.Errorf("unknown @git subcommand: %s", in.Subcommand)
}

func (r *GitAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	if in.Subcommand == "" {
		return r.Handle(ctx, in, nil)
	}

	prompt, err := r.getSystemPrompt(in)
	if err != nil {
		return nil, err
	}
	model := internal.Level1(r.config.LLM)
	content, err := llm.Send(ctx, r.Role, prompt, model, in.Input())
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   "GIT",
		Content: content,
	}, nil
}

func (r *GitAgent) Handle(ctx context.Context, req *api.Request, next api.HandlerNext) (*api.Response, error) {
	// let LLM decide which subcommand to call based on the user input
	var clip = req.Clip()
	prompt := resource.GetCliGitSystem()

	action := func(ctx context.Context, sub string) (string, error) {
		log.Debugf("action: GIT subcommand: %s\n", sub)
		req.Subcommand = sub
		resp, err := r.Send(ctx, req)
		if err != nil {
			return "", err
		}
		return resp.Content, nil
	}

	model := internal.Level1(r.config.LLM)
	model.Tools = llm.GetGitTools()

	msg := &internal.Message{
		Role:   "system",
		Prompt: prompt,
		Model:  model,
		Input:  clip,
		Next:   action,
	}
	resp, err := llm.Chat(ctx, msg)
	if err != nil {
		return nil, err
	}
	return &api.Response{
		Content: resp.Content,
	}, nil
}
