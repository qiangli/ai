package agent

import (
	"context"

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
	return resource.GetPrSystem(baseCommand(in.Subcommand))
}

func (r *PrAgent) format(in *UserInput, resp string) (string, error) {
	return resource.FormatPrResponse(baseCommand(in.Subcommand), resp)
}

func (r *PrAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	log.Debugf("PrAgent.Send: subcommand: %s\n", in.Subcommand)

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

	log.Debugf("PrAgent.Send: model: %+v\n", model)

	resp, err := llm.Send(ctx, r.Role, prompt, model, input)
	if err != nil {
		return nil, err
	}

	log.Debugf("PrAgent.Send response: %v", resp)

	// convert json response to markdown
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
	intent := req.Intent()
	if intent == "" {
		log.Debugf("PrAgent.Handle: no intent, default to: describe\n")
		req.Subcommand = "describe"
		return r.Send(ctx, req)
	}

	// let LLM decide which subcommand to call based on the user input
	prompt := resource.GetCliPrSubSystem()

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
		Input:  intent,
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
