package agent

// import (
// 	"context"

// 	"github.com/qiangli/ai/internal"
// 	"github.com/qiangli/ai/internal/llm"
// )

// type EvalAgent struct {
// 	config *internal.AppConfig

// 	Role   string
// 	Prompt string
// }

// func NewEvalAgent(cfg *internal.AppConfig) (*EvalAgent, error) {
// 	role := cfg.Role
// 	prompt := cfg.Prompt

// 	if role == "" {
// 		role = "system"
// 	}

// 	agent := EvalAgent{
// 		config: cfg,
// 		Role:   role,
// 		Prompt: prompt,
// 	}
// 	return &agent, nil
// }

// func (r *EvalAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
// 	model := internal.CreateModel(r.config.LLM)
// 	req := &internal.Message{
// 		Role:   r.Role,
// 		Prompt: r.Prompt,
// 		Model:  model,
// 		Input:  in.Query(),
// 	}

// 	resp, err := llm.Chat(ctx, req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &ChatMessage{
// 		Agent:   "EVAL",
// 		Content: resp.Content,
// 	}, nil
// }
