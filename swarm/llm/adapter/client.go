package adapter

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/anthropic"
	"github.com/qiangli/ai/swarm/llm/gemini"
	"github.com/qiangli/ai/swarm/llm/openai"
)

type adapters struct {
}

func (r *adapters) Get(key string) (llm.LLMAdapter, error) {
	if v, ok := adapterRegistry[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("LLM adapter %q not found", key)
}

var defaultAdapters = &adapters{}

var adapterRegistry map[string]llm.LLMAdapter

func init() {
	adapterRegistry = make(map[string]llm.LLMAdapter)
	adapterRegistry["chat"] = &ChatAdapter{}
	adapterRegistry["text"] = &ResponseAdapter{}
	adapterRegistry["response"] = &ResponseAdapter{}
	adapterRegistry["image"] = &ImageAdapter{}
	adapterRegistry["tts"] = &TtsAdapter{}
	adapterRegistry["audio"] = &AudioAdapter{}
	adapterRegistry["video"] = &VideoAdapter{}
}

func GetAdapters() llm.AdapterRegistry {
	return defaultAdapters
}

type ChatAdapter struct{}

func (r *ChatAdapter) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
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
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}
	return resp, nil
}

type ImageAdapter struct{}

func (r *ImageAdapter) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
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
		resp, err = openai.Image(ctx, req)
	case "anthropic":
		return nil, fmt.Errorf("Not supported: %s", provider)
	default:
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}
	return resp, nil
}

type ResponseAdapter struct{}

func (r *ResponseAdapter) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
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
		resp, err = openai.SendV3(ctx, req)
	case "anthropic":
		return nil, fmt.Errorf("Not supported: %s", provider)
	default:
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}
	return resp, nil
}

type TtsAdapter struct{}

func (r *TtsAdapter) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
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
		resp, err = openai.TTS(ctx, req)
	case "anthropic":
		return nil, fmt.Errorf("Not supported: %s", provider)
	default:
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}
	return resp, nil
}

type AudioAdapter struct{}

func (r *AudioAdapter) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
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
		resp, err = openai.Audio(ctx, req)
	case "anthropic":
		return nil, fmt.Errorf("Not supported: %s", provider)
	default:
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}
	return resp, nil
}

type VideoAdapter struct{}

func (r *VideoAdapter) Call(ctx context.Context, req *llm.Request) (*llm.Response, error) {
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
		resp, err = openai.Video(ctx, req)
	case "anthropic":
		return nil, fmt.Errorf("Not supported: %s", provider)
	default:
		return nil, fmt.Errorf("Unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("No response")
	}
	return resp, nil
}
