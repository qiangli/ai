package openai

import (
	"context"
	"io"

	"github.com/openai/openai-go/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
)

func TTS(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	log.GetLogger(ctx).Debugf(">OPENAI:\n tts req: %+v\n", req)

	var err error
	var resp *llm.Response

	resp, err = tts(ctx, req)

	log.GetLogger(ctx).Debugf(">OPENAI:\n tts resp: %+v err: %v\n", resp, err)
	return resp, err
}

func tts(ctx context.Context, req *llm.Request) (*llm.Response, error) {
	client, err := NewClient(req.Model, req.Vars)
	if err != nil {
		return nil, err
	}

	// messages := make([]string, 0)
	// for _, v := range req.Messages {
	// 	messages = append(messages, v.Content)
	// }
	// prompt := strings.Join(messages, "\n")

	result, err := client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Model:          req.Model.Model,
		Instructions:   openai.String(req.Instruction),
		Input:          req.Query,
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatPCM,
		Voice:          openai.AudioSpeechNewParamsVoiceAlloy,
	})
	if err != nil {
		return nil, err
	}
	defer result.Body.Close()

	b, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	return &llm.Response{
		Result: &api.Result{
			Value: string(b),
		},
	}, nil
}
