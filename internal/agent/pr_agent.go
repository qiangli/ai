package agent

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal"
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
	switch in.Subcommand {
	case "":
		return resource.GetPrDescriptionSystem()
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
	switch in.Subcommand {
	case "":
		return resource.FormatPrDescription(resp)
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

	resp, err := llm.Send(r.config.LLM, ctx, r.Role, prompt, input)
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
