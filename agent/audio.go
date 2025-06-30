package agent

import (
	"context"
	"os"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/qiangli/ai/swarm/api"
)

func transcribe(cfg *api.AppConfig, audiofile string) (string, error) {
	client := openai.NewClient(
		option.WithAPIKey(cfg.TTS.ApiKey),
		// option.WithBaseURL(baseUrl),
	)
	ctx := context.Background()

	file, err := os.Open(audiofile)
	if err != nil {
		return "", err
	}

	transcription, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
		Model: openai.AudioModelWhisper1,
		File:  file,
	})
	if err != nil {
		return "", err
	}

	return transcription.Text, nil
}
