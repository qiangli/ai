package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/resource/pr"
)

type PrAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewPrAgent(cfg *internal.AppConfig) (*PrAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}
	agent := PrAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *PrAgent) getSystemPrompt(in *UserInput) (string, error) {
	if r.Prompt != "" {
		return r.Prompt, nil
	}
	switch baseCommand(in.Subcommand) {
	case "describe":
		return resource.GetPrDescriptionSystem()
	case "review":
		return resource.GetPrReviewSystem()
	case "improve":
		return resource.GetPrCodeSystem()
	case "changelog":
		return resource.GetPrChangelogSystem(), nil
	}
	return "", fmt.Errorf("unknown @pr subcommand: %s", in.Subcommand)
}

func (r *PrAgent) format(in *UserInput, resp string) (string, error) {
	switch baseCommand(in.Subcommand) {
	case "describe":
		return resource.FormatPrDescription(resp)
	case "review":
		return resource.FormatPrReview(resp)
	case "improve":
		return resource.FormatPrCodeSuggestion(resp)
	case "changelog":
		return resource.FormatPrChangelog(resp)
	}
	return "", fmt.Errorf("unknown subcommand: %s", in.Subcommand)
}

func (r *PrAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	if in.Subcommand == "" {
		return r.Handle(ctx, in, nil)
	}

	input, err := resource.GetPrUser(&pr.Input{
		Instruction: in.Message,
		Diff:        in.Content,
	})
	if err != nil {
		return nil, err
	}

	prompt, err := r.getSystemPrompt(in)
	if err != nil {
		return nil, err
	}
	model := internal.Level1(r.config.LLM)
	resp, err := llm.Send(ctx, r.Role, prompt, model, input)
	if err != nil {
		return nil, err
	}

	log.Debugf("PR response: %v", resp)

	content, err := r.format(in, resp)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Agent:   in.Agent,
		Content: content,
	}, nil
}

func (r *PrAgent) Handle(ctx context.Context, req *api.Request, next api.HandlerNext) (*api.Response, error) {
	// let LLM decide which subcommand to call based on the user input
	var clip = req.Clip()
	prompt := resource.GetCliPrSystem()

	action := func(ctx context.Context, sub string) (string, error) {
		log.Debugf("action: PR subcommand: %s\n", sub)
		req.Subcommand = sub
		resp, err := r.Send(ctx, req)
		if err != nil {
			return "", err
		}
		return resp.Content, nil
	}

	model := internal.Level1(r.config.LLM)
	model.Tools = llm.GetPrTools()

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
