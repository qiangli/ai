package agent

import (
	"context"
	"encoding/json"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

// HelpAgent auto selects the best agent to handle the user query
type HelpAgent struct {
	config *internal.AppConfig

	Role    string
	Message string
}

func NewHelpAgent(cfg *internal.AppConfig) (*HelpAgent, error) {
	role := cfg.Role
	content := cfg.Prompt

	if role == "" {
		role = "system"
	}
	// if content == "" {
	// 	content = resource.GetCliAgentDetectSystem()
	// }

	agent := HelpAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

type AgentDetect struct {
	Agent   string `json:"agent"`
	Command string `json:"command"`
}

func (r *HelpAgent) Handle(ctx context.Context, req *api.Request, next api.HandlerNext) (*api.Response, error) {
	model := internal.Level1(r.config.LLM)
	// model.Tools = llm.GetAIHelpTools()
	msg := &internal.Message{
		Role:   r.Role,
		Prompt: r.Message,
		Model:  model,
		Input:  req.Query(),
	}
	resp, err := llm.Chat(ctx, msg)
	if err != nil {
		return nil, err
	}

	var data AgentDetect
	if err := json.Unmarshal([]byte(resp.Content), &data); err != nil {
		// better continue the conversation than err
		log.Debugf("failed to unmarshal content: %v\n", err)
		data = AgentDetect{
			Agent:   "ask",
			Command: "",
		}
	}

	log.Debugf("dispatching: %+v\n", data)

	req.Agent = data.Agent
	req.Subcommand = data.Command

	return next(ctx, req)
}
