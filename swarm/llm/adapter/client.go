package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm/anthropic"
	"github.com/qiangli/ai/swarm/llm/gemini"
	"github.com/qiangli/ai/swarm/llm/openai"
	"github.com/qiangli/ai/swarm/llm/xai"
)

const MaxTurnsLimit = 100
const MaxTimeLimit = 900 // 15 min

const DefaultMaxTurns = 50
const DefaultMaxTime = 600 // 10 min

// const DefaultMaxSpan = 1440 // 24 hours
// const DefaultMaxHistory = 3

type adapters struct {
}

func (r *adapters) Get(key string) (api.LLMAdapter, error) {
	if v, ok := adapterRegistry[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("LLM adapter %q not found", key)
}

var defaultAdapters = &adapters{}

var adapterRegistry map[string]api.LLMAdapter

func init() {
	adapterRegistry = make(map[string]api.LLMAdapter)
	adapterRegistry["echo"] = &EchoAdapter{}
	adapterRegistry["chat"] = &ChatAdapter{}
	adapterRegistry["text"] = &TextAdapter{}
	// adapterRegistry["response"] = &ResponseAdapter{}
	adapterRegistry["image"] = &ImageAdapter{}
	adapterRegistry["tts"] = &TtsAdapter{}
	adapterRegistry["audio"] = &AudioAdapter{}
	adapterRegistry["video"] = &VideoAdapter{}
}

func GetAdapters() api.AdapterRegistry {
	return defaultAdapters
}

type EchoAdapter struct{}

func (r *EchoAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var resp api.Response

	// custom response: echo__id
	if len(req.Arguments) > 0 {
		id := req.Arguments.Kitname().ID()
		if v, found := req.Arguments["echo__"+id]; found {
			resp.Result = &api.Result{
				Value: api.ToString(v),
			}
			return &resp, nil
		}
	}

	// resolve cycles
	var agent *api.Agent
	if len(req.Arguments) > 0 {
		v, found := req.Arguments["agent"]
		if found {
			if a, ok := v.(*api.Agent); ok {
				agent = a
				req.Arguments["agent"] = api.NewPackname(a.Pack, a.Name)
			}
		}
	}

	v, err := json.Marshal(req)

	if agent != nil {
		req.Arguments["agent"] = agent
	}

	if err != nil {
		resp.Result = &api.Result{
			Value: err.Error(),
		}
	} else {
		resp.Result = &api.Result{
			Value: string(v),
		}
	}
	return &resp, nil
}

type ChatAdapter struct{}

func (r *ChatAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var err error
	var resp *api.Response

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
	case "xai":
		resp, err = xai.Send(ctx, req)
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

func (r *ImageAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var err error
	var resp *api.Response

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

type TextAdapter struct{}

func (r *TextAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var err error
	var resp *api.Response

	if req.Model == nil {
		return nil, fmt.Errorf("No LLM model provided")
	}

	provider := req.Model.Provider
	//
	switch provider {
	case "gemini":
		// return nil, fmt.Errorf("Not supported: %s", provider)
		// TODO not working
		// https://developers.googleblog.com/en/gemini-is-now-accessible-from-the-openai-library/
		// https://generativelanguage.googleapis.com/v1beta/openai/
		// resp, err = openai.Send(ctx, req)
		resp, err = gemini.Send(ctx, req)
	case "openai":
		// new response api
		resp, err = openai.SendV3(ctx, req)
	case "anthropic":
		resp, err = anthropic.Send(ctx, req)
	case "xai":
		resp, err = xai.Send(ctx, req)
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

func (r *TtsAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var err error
	var resp *api.Response

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

func (r *AudioAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var err error
	var resp *api.Response

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

func (r *VideoAdapter) Call(ctx context.Context, req *api.Request) (*api.Response, error) {
	var err error
	var resp *api.Response

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
