package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm/anthropic"
	"github.com/qiangli/ai/swarm/llm/gemini"
	"github.com/qiangli/ai/swarm/llm/openai"
)

func Send(ctx context.Context, req *api.LLMRequest) (*api.LLMResponse, error) {
	log.Debugf(">>>LLM Client:\n Model: %s Model: %+v, Messages: %v Tools: %v\n\n", req.Model, req.Model, len(req.Messages), len(req.Tools))

	var err error
	var resp *api.LLMResponse

	if req.Model == nil {
		return nil, fmt.Errorf("No LLM model provided")
	}

	provider := req.Model.Provider
	model := req.Model.Model()

	//
	if provider == "" {
		provider = "openai"
		switch {
		case strings.HasPrefix(model, "gemini-"):
			provider = "gemini"
		case strings.HasPrefix(model, "claude-"):
			provider = "anthropic"
		default:
			log.Debugf("model provider is unknown, assuming openai")
		}
	}

	switch provider {
	case "gemini":
		// TODO not working
		// https://developers.googleblog.com/en/gemini-is-now-accessible-from-the-openai-library/
		// https://generativelanguage.googleapis.com/v1beta/openai/
		// resp, err = openai.Send(ctx, req)
		resp, err = gemini.Send(ctx, req)
	case "openai":
		resp, err = openai.Send(ctx, req)
	case "anthropic":
		resp, err = anthropic.Send(ctx, req)
	default:
		resp, err = openai.Send(ctx, req)
	}

	if err != nil {
		log.Errorf("***LLM Client: %s\n\n", err)
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}

	log.Debugf("<<<LLM Client:\n Content type: %s Content: %v\n\n", resp.ContentType, len(resp.Content))
	return resp, nil
}
