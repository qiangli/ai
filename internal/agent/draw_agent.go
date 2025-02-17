package agent

// import (
// 	"context"

// 	"github.com/qiangli/ai/internal"
// 	"github.com/qiangli/ai/internal/api"
// 	"github.com/qiangli/ai/internal/llm"
// )

// type DrawAgent struct {
// 	config *internal.AppConfig

// 	Role   string
// 	Prompt string
// }

// func NewDrawAgent(cfg *internal.AppConfig) (*DrawAgent, error) {
// 	role := cfg.Role
// 	prompt := cfg.Prompt

// 	if role == "" {
// 		role = "system"
// 	}

// 	agent := DrawAgent{
// 		config: cfg,
// 		Role:   role,
// 		Prompt: prompt,
// 	}
// 	return &agent, nil
// }

// func (r *DrawAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
// 	model := internal.ImageModel(r.config.LLM)
// 	req := &internal.Message{
// 		Role:   r.Role,
// 		Prompt: r.Prompt,
// 		Model:  model,
// 		Input:  in.Query(),
// 	}

// 	resp, err := llm.GenerateImage(ctx, req)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &ChatMessage{
// 		Agent:       "DRAW",
// 		ContentType: api.ContentTypeB64JSON,
// 		Content:     resp.Content,
// 	}, nil
// }
