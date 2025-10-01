package adapter

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/anthropic"
	"github.com/qiangli/ai/swarm/llm/gemini"
	"github.com/qiangli/ai/swarm/llm/openai"
)

type adapters struct{}

func (r *adapters) Get(key string) (llm.LLMAdapter, error) {
	if v, ok := adapterRegistry[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("LLM adapter %q not found", key)
}

var adapterRegistry map[string]llm.LLMAdapter

func init() {
	adapterRegistry = make(map[string]llm.LLMAdapter)
	adapterRegistry["chat"] = Chat
	adapterRegistry["image-gen"] = ImageGen
}

var defaultAdapters = &adapters{}

func GetAdapters() llm.AdapterRegistry {
	return defaultAdapters
}

func Chat(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	// log.GetLogger(ctx).Debugf(">LLM Chat:\n %v\n", req)

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
		// log.GetLogger(ctx).Errorf("***LLM Client: %s\n", err)
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}

	// log.GetLogger(ctx).Debugf(">LLM Chat:\n Content type: %s Content: %v\n", resp.ContentType, len(resp.Content))
	return resp, nil
}

func ImageGen(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	// log.GetLogger(ctx).Debugf(">LLM ImageGen:\n Model: %s Model: %+v, Messages: %v Tools: %v\n", req.Model, req.Model, len(req.Messages), len(req.Tools))

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
		// log.GetLogger(ctx).Errorf("***LLM Client: %s\n", err)
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}

	// log.GetLogger(ctx).Debugf(">LLM ImageGen:\n Content type: %s Content: %v\n", resp.ContentType, len(resp.Content))
	return resp, nil
}
