package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/shared/constant"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/api/model"
)

// fake openai chat completion
func fake(
	req *http.Request,
	model *model.Model,
	vars *api.Vars,
) (*http.Response, error) {
	chatCompletion := dummyChatCompletion(model, vars)
	jsonData, err := json.Marshal(chatCompletion)
	if err != nil {
		return nil, err
	}

	// Create a fake HTTP response
	headers := make(http.Header)
	headers.Set("Content-type", "application/json")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(jsonData)),
		Request:    req,
		Header:     headers,
	}

	return resp, nil
}

// dummyChatCompletion createa a dummy ChatCompletion
func dummyChatCompletion(model *model.Model, vars *api.Vars) *openai.ChatCompletion {
	return &openai.ChatCompletion{
		ID: uuid.NewString(),
		Choices: []openai.ChatCompletionChoice{
			dummyChatCompletionChoice(vars.Config.DryRunContent),
		},
		Created:           time.Now().Unix(),
		Model:             model.Model(),
		Object:            constant.ChatCompletion("chat.completion"),
		ServiceTier:       openai.ChatCompletionServiceTier("auto"),
		SystemFingerprint: "dummy-fingerprint",
		Usage:             openai.CompletionUsage{ /* Initialize with dummy usage */ },
	}
}

// dummyChatCompletionChoice creates a ChatCompletionChoice filled with dummy values.
func dummyChatCompletionChoice(content string) openai.ChatCompletionChoice {
	return openai.ChatCompletionChoice{
		FinishReason: "stop",
		Index:        0,
		Logprobs:     openai.ChatCompletionChoiceLogprobs{},
		Message:      dummyChatCompletionMessage(content),
		// JSON: struct {
		// 	FinishReason respjson.Field
		// 	Index        respjson.Field
		// 	Logprobs     respjson.Field
		// 	Message      respjson.Field
		// 	ExtraFields  map[string]respjson.Field
		// 	raw          string
		// }{
		// 	FinishReason: respjson.Field{Valid: true},
		// 	Index:        respjson.Field{Valid: true},
		// 	Logprobs:     respjson.Field{Valid: true},
		// 	Message:      respjson.Field{Valid: true},
		// 	ExtraFields:  make(map[string]respjson.Field),
		// 	raw:          "{}",
		// },
	}
}

// dummyChatCompletionMessage returns a ChatCompletionMessage populated with dummy values.
func dummyChatCompletionMessage(content string) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Content: content,
		// Refusal:    "dummy refusal",
		Role:        constant.Assistant("assistant"),
		Annotations: []openai.ChatCompletionMessageAnnotation{
			// Add dummy annotations if necessary
		},
		Audio: openai.ChatCompletionAudio{
			// Initialize with dummy audio data
		},
		// FunctionCall: openai.ChatCompletionMessageFunctionCall{
		// 	// Initialize with dummy function call data
		// },
		ToolCalls: []openai.ChatCompletionMessageToolCall{},
		// JSON: struct {
		// 	Content      respjson.Field
		// 	Refusal      respjson.Field
		// 	Role         respjson.Field
		// 	Annotations  respjson.Field
		// 	Audio        respjson.Field
		// 	FunctionCall respjson.Field
		// 	ToolCalls    respjson.Field
		// 	ExtraFields  map[string]respjson.Field
		// 	raw          string
		// }{
		// 	Content:      respjson.Field{Valid: true},
		// 	Refusal:      respjson.Field{Valid: true},
		// 	Role:         respjson.Field{Valid: true},
		// 	Annotations:  respjson.Field{Valid: false},
		// 	Audio:        respjson.Field{Valid: false},
		// 	FunctionCall: respjson.Field{Valid: false},
		// 	ToolCalls:    respjson.Field{Valid: false},
		// 	ExtraFields:  make(map[string]respjson.Field),
		// 	raw:          "dummy raw data",
		// },
	}
}
