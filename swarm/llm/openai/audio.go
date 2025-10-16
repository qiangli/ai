package openai

import (
	"context"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

func Audio(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n audio req: %+v\n", req)

	var err error
	var resp *llm.Response

	resp, err = transcribe(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n audio resp: %+v err: %v\n", resp, err)
	return resp, err
}

func transcribe(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClient(req.Model, req.Vars)
	if err != nil {
		return nil, err
	}

	messages := make([]string, 0)
	for _, v := range req.Messages {
		messages = append(messages, v.Content)
	}

	prompt := strings.Join(messages, "\n")

	var filename string
	if req.Arguments != nil {
		if file, found := req.Arguments["file"]; found {
			if v, ok := file.(string); ok {
				filename = v
			}
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	transcription, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		Model:  req.Model.Model,
		File:   file,
		Prompt: openai.String(prompt),
	})
	if err != nil {
		return nil, err
	}

	return &llm.Response{
		Result: &api.Result{
			Value: transcription.Text,
		},
	}, nil
}
