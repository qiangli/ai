package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/anthropic"
	"github.com/qiangli/ai/swarm/llm/gemini"
	"github.com/qiangli/ai/swarm/llm/openai"
	"github.com/qiangli/ai/swarm/log"
)

func Chat(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debug(">>>LLM Chat:\n Model: %s Model: %+v, Messages: %v Tools: %v\n\n", req.Model, req.Model, len(req.Messages), len(req.Tools))

	var err error
	var resp *llm.Response

	if req.Model == nil {
		return nil, fmt.Errorf("No LLM model provided")
	}

	provider := req.Model.Provider

	//
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
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		log.GetLogger(ctx).Error("***LLM Client: %s\n\n", err)
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}

	log.GetLogger(ctx).Debug("<<<LLM Chat:\n Content type: %s Content: %v\n\n", resp.ContentType, len(resp.Content))
	return resp, nil
}

func ImageGen(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debug(">>>LLM ImageGen:\n Model: %s Model: %+v, Messages: %v Tools: %v\n\n", req.Model, req.Model, len(req.Messages), len(req.Tools))

	var err error
	var resp *llm.Response

	if req.Model == nil {
		return nil, fmt.Errorf("No LLM model provided")
	}

	provider := req.Model.Provider
	//
	switch provider {
	case "gemini":
		return nil, fmt.Errorf("Not supported: %s", provider)
	case "openai":
		resp, err = openai.ImageGen(ctx, req)
	case "anthropic":
		return nil, fmt.Errorf("Not supported: %s", provider)
	default:
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		log.GetLogger(ctx).Error("***LLM Client: %s\n\n", err)
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}

	log.GetLogger(ctx).Debug("<<<LLM ImageGen:\n Content type: %s Content: %v\n\n", resp.ContentType, len(resp.Content))
	return resp, nil
}
