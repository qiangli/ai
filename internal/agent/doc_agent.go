package agent

import (
	"context"
	"os"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/resource"
)

type DocAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string

	Template string
}

func NewDocAgent(cfg *internal.AppConfig) (*DocAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}

	agent := DocAgent{
		config:   cfg,
		Role:     role,
		Prompt:   prompt,
		Template: cfg.Template,
	}
	return &agent, nil
}

func (r *DocAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	model := internal.Level2(r.config.LLM)

	// read the template
	var temp []byte
	if r.Template == "" {
		return nil, internal.NewUserInputError("no template file provided")
	}
	var err error
	temp, err = os.ReadFile(r.Template)
	if err != nil {
		return nil, err
	}
	if len(temp) == 0 {
		return nil, internal.NewUserInputError("empty template file")
	}

	// read the draft
	draft, err := in.FileContent()
	if err != nil {
		return nil, err
	}
	if len(draft) == 0 {
		return nil, internal.NewUserInputError("empty draft content")
	}

	prompt, err := resource.GetDocComposeSystem(&resource.DocCompose{
		Template: string(temp),
		Draft:    string(draft),
	})
	if err != nil {
		return nil, err
	}

	req := &internal.Message{
		Role:   r.Role,
		Prompt: prompt,
		Model:  model,
		Input:  in.Query(),
	}

	resp, err := llm.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	content := resp.Content
	return &ChatMessage{
		Agent:   "DOC",
		Content: content,
	}, nil
}
