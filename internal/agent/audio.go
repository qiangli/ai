package agent

// import (
// 	"context"
// 	"os"

// 	"github.com/openai/openai-go"
// 	"github.com/openai/openai-go/option"

// 	"github.com/qiangli/ai/swarm/api"
// 	"github.com/qiangli/ai/swarm/llm"
// )

// func transcribe(cfg *api.AppConfig, audiofile string) (string, error) {
// 	m, err := cfg.ModelLoader(llm.TTS)
// 	if err != nil {
// 		return "", err
// 	}
// 	client := openai.NewClient(
// 		option.WithAPIKey(m.ApiKey),
// 		// option.WithBaseURL(baseUrl),
// 	)
// 	ctx := context.Background()

// 	file, err := os.Open(audiofile)
// 	if err != nil {
// 		return "", err
// 	}

// 	transcription, err := client.Audio.Transcriptions.New(ctx, openai.AudioTranscriptionNewParams{
// 		Model: openai.AudioModelWhisper1,
// 		File:  file,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	return transcription.Text, nil
// }
